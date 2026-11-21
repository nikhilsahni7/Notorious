import { FestiveCelebration } from "@/components/FestiveCelebration";
import { PWARegistrar } from "@/components/PWARegistrar";
import { WhatsAppButton } from "@/components/WhatsAppButton";
import { AuthProvider } from "@/providers/AuthProvider";
import { ChatProviderWrapper } from "@/providers/ChatProviderWrapper";
import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Knotorious Search",
  description: "Advanced database search system",
  manifest: "/manifest.webmanifest",
  icons: { icon: "/favicon.ico", apple: "/knotorious-apple-icon.png" },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <AuthProvider>
          <ChatProviderWrapper>
            <FestiveCelebration />
            <PWARegistrar />
            {children}
            <WhatsAppButton />
          </ChatProviderWrapper>
        </AuthProvider>
      </body>
    </html>
  );
}
