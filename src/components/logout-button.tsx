"use client";

import { useRouter } from "next/navigation";
import { Button } from "@base-ui/react/button";
import { authClient } from "@/lib/auth-client";

export function LogoutButton({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  const router = useRouter();

  async function handleLogout() {
    await authClient.signOut();
    router.push("/");
  }

  return (
    <Button onClick={handleLogout} className={className}>
      {children}
    </Button>
  );
}
