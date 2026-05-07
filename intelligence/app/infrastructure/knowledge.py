import asyncio
import logging

import psycopg
from psycopg.rows import dict_row

from ..config import Settings

log = logging.getLogger(__name__)


class KnowledgeRepository:
    def __init__(self, settings: Settings):
        self.settings = settings
        self.model = None
        self.mode = "lexical"
        self.error = None

    async def initialize(self):
        if self.settings.retrieval_mode == "lexical":
            return
        try:
            from fastembed import TextEmbedding

            self.model = await asyncio.to_thread(
                TextEmbedding,
                model_name="sentence-transformers/all-MiniLM-L6-v2",
                cache_dir="/root/.cache/fastembed",
                threads=2,
            )
            await self.index_missing()
            self.mode = "vector"
        except Exception as exc:
            self.error = str(exc)
            log.exception("Embedding model unavailable; using explicit lexical fallback")

    async def index_missing(self):
        if not self.model:
            return
        async with await psycopg.AsyncConnection.connect(
            self.settings.database_url, row_factory=dict_row
        ) as conn:
            rows = await (
                await conn.execute(
                    "SELECT id,content FROM knowledge WHERE branch_id=%s AND embedding IS NULL",
                    (self.settings.branch_id,),
                )
            ).fetchall()
            if not rows:
                return
            vectors = await asyncio.to_thread(
                lambda: list(self.model.embed([r["content"] for r in rows]))
            )
            for row, vector in zip(rows, vectors):
                await conn.execute(
                    "UPDATE knowledge SET embedding=%s::vector WHERE id=%s AND branch_id=%s",
                    (str(vector.tolist()), row["id"], self.settings.branch_id),
                )

    async def search(self, text: str):
        async with await psycopg.AsyncConnection.connect(
            self.settings.database_url, row_factory=dict_row
        ) as conn:
            if self.mode == "vector":
                vector = await asyncio.to_thread(lambda: list(self.model.embed([text]))[0].tolist())
                rows = await (
                    await conn.execute(
                        "SELECT source,content,1-(embedding <=> %s::vector) AS score FROM knowledge WHERE branch_id=%s AND embedding IS NOT NULL ORDER BY embedding <=> %s::vector LIMIT 3",
                        (str(vector), self.settings.branch_id, str(vector)),
                    )
                ).fetchall()
            else:
                rows = await (
                    await conn.execute(
                        "SELECT source,content,ts_rank_cd(to_tsvector('english',content),websearch_to_tsquery('english',%s)) AS score FROM knowledge WHERE branch_id=%s ORDER BY score DESC LIMIT 3",
                        (text, self.settings.branch_id),
                    )
                ).fetchall()
            return {"mode": self.mode, "documents": rows}

    async def remember(self, incident_id, content):
        async with await psycopg.AsyncConnection.connect(self.settings.database_url) as conn:
            await conn.execute(
                "INSERT INTO knowledge(branch_id,source,content) VALUES(%s,%s,%s) ON CONFLICT(branch_id,source) DO NOTHING",
                (self.settings.branch_id, f"resolved-incident:{incident_id}", content),
            )
        await self.index_missing()
