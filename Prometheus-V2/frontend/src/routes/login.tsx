import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { LoginForm } from "@/components/auth/login-form";

export const Route = createFileRoute("/login")({
  component: LoginRoute,
});

function LoginRoute() {
  const navigate = useNavigate();
  return (
    <div className="min-h-full flex items-center justify-center p-4">
      <LoginForm onSuccess={() => navigate({ to: "/" })} />
    </div>
  );
}
