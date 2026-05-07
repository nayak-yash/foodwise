import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Foodwise — Operations intelligence",
  description: "Evidence-backed restaurant investigations, approvals and verified recovery.",
};
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
