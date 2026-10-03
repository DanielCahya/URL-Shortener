import { useState } from "react";
import { useParams } from "react-router-dom";
import { GlassCard } from "../components/ui/GlassCard";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";
import { Lock } from "lucide-react";

export function PasswordWall() {
  const { alias } = useParams();
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      // Create a form to POST to the original backend endpoint that handles the password verification and redirect
      // Since the backend handles the redirect via a 302, we can just submit a native form
      const form = document.createElement('form');
      form.method = 'POST';
      form.action = `http://localhost:8080/${alias}`;
      
      const pwdInput = document.createElement('input');
      pwdInput.type = 'hidden';
      pwdInput.name = 'password';
      pwdInput.value = password;
      
      form.appendChild(pwdInput);
      document.body.appendChild(form);
      form.submit();
      
      // Note: If the password is wrong, the backend will return a 401/403 page.
      // A more robust React approach would be to send an API request to verify, then redirect.
      // But for now, we rely on the backend's standard behavior.
    } catch (err) {
      setError("Failed to verify password");
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-[70vh] px-4">
      <GlassCard className="w-full max-w-sm text-center">
        <div className="w-16 h-16 bg-surfaceHighlight/50 rounded-full flex items-center justify-center mx-auto mb-6">
          <Lock size={32} className="text-primary" />
        </div>
        
        <h2 className="text-2xl font-bold text-textMain mb-2">Protected Link</h2>
        <p className="text-textMuted mb-8">This link requires a password to access.</p>
        
        <form onSubmit={handleSubmit} className="space-y-4 text-left">
          <Input 
            type="password" 
            placeholder="Enter password"
            icon={Lock}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoFocus
          />
          {error && <div className="text-red-400 text-sm">{error}</div>}
          <Button type="submit" variant="primary" className="w-full" isLoading={loading}>
            Unlock Link
          </Button>
        </form>
      </GlassCard>
    </div>
  );
}
