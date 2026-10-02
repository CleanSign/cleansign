import type { Metadata } from "next";
import { Gothic_A1 } from "next/font/google";
import "./globals.css";

const gothic = Gothic_A1({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800", "900"],
  variable: "--font-gothic",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Clean Sign",
  description: "계약서의 위험 조항을 미리 확인하세요",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="ko" className={gothic.variable}>
      <body className="font-sans antialiased">{children}</body>
    </html>
  );
}
