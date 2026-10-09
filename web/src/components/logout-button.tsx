"use client";

import { useRouter } from "next/navigation";
import { Button } from "@base-ui/react/button";

export function LogoutButton({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  const router = useRouter();

  async function handleLogout() {
    await fetch("/api/v1/auth/logout", { method: "POST" }).catch(() => null);
    router.push("/");
  }

  return (
    <Button onClick={handleLogout} className={className}>
      {children}
    </Button>
  );
}
