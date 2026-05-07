export const formatLabel = (text: string) => text.replaceAll("_", " ");

export const formatTime = (text: string) =>
  new Date(text).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
