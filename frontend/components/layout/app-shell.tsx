import {
  Bell,
  Boxes,
  ChartNoAxesCombined,
  ChevronDown,
  LayoutDashboard,
  Search,
  ShieldCheck,
  Sparkles,
  UtensilsCrossed,
} from "lucide-react";
import type { AgentStatus, View } from "@/lib/types";

const navigation = [
  { id: "overview", name: "Overview", icon: LayoutDashboard },
  { id: "incidents", name: "Investigations", icon: Search },
  { id: "inventory", name: "Inventory & supply", icon: Boxes },
  { id: "forecasts", name: "Demand planning", icon: ChartNoAxesCombined },
] as const;

interface Props {
  view: View;
  onNavigate: (view: View) => void;
  activeIncidents: number;
  agent: AgentStatus | null;
  children: React.ReactNode;
}

export function AppShell({ view, onNavigate, activeIncidents, agent, children }: Props) {
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <a className="brand" href="/" aria-label="Foodwise home">
          <span className="brand-mark">
            <UtensilsCrossed size={21} />
          </span>
          <span>
            food<span className="brand-light">wise</span>
            <small>OPERATIONS INTELLIGENCE</small>
          </span>
        </a>
        <div className="workspace">
          <span className="workspace-avatar">DK</span>
          <span>
            Downtown Kitchen<small>Restaurant workspace</small>
          </span>
          <ChevronDown size={14} />
        </div>
        <div className="nav-label">WORKSPACE</div>
        <nav>
          {navigation.map(({ id, name, icon: Icon }) => (
            <button
              key={id}
              aria-label={name}
              className={view === id ? "nav-item selected" : "nav-item"}
              onClick={() => onNavigate(id)}
            >
              <Icon size={18} />
              {name}
              {id === "incidents" && activeIncidents > 0 && (
                <span className="nav-count">{activeIncidents}</span>
              )}
            </button>
          ))}
        </nav>
        <div className="sidebar-note">
          <Sparkles size={20} />
          <strong>Your operations, connected.</strong>
          <p>From the first signal to verified recovery. Every decision backed by evidence.</p>
          <div className="agent-live">
            <span className={agent?.ready ? "live-dot" : "live-dot idle"} />
            {agent?.ready ? "Investigator online" : "Connecting investigator"}
          </div>
        </div>
        <div className="profile">
          <span className="profile-avatar">RM</span>
          <span>
            Restaurant manager<small>Demo workspace</small>
          </span>
          <ShieldCheck size={17} />
        </div>
      </aside>
      <main>
        <header className="topbar">
          <div>
            <span className="breadcrumb">Foodwise</span>
            <span className="slash">/</span>
            {navigation.find((item) => item.id === view)?.name}
          </div>
          <div className="topbar-right">
            <span className="environment">
              <span className="live-dot" />
              LOCAL DEMO
            </span>
            <span className="topbar-divider" />
            <Bell size={18} />
            <span className="user-dot">RM</span>
          </div>
        </header>
        <div className="content">
          {children}
          <footer className="page-footer">
            <span>
              <UtensilsCrossed size={14} />
              Foodwise
            </span>
            <span>Evidence first. People in control.</span>
            <span>Go · LangGraph · Kafka · pgvector</span>
          </footer>
        </div>
      </main>
    </div>
  );
}
