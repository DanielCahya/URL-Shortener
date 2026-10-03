import { BrowserRouter as Router, Routes, Route } from "react-router-dom";
import { Navbar } from "./components/ui/Navbar";
import { Landing } from "./pages/Landing";
import { Dashboard } from "./pages/Dashboard";
import { AuthPage } from "./pages/AuthPage";
import { Analytics } from "./pages/Analytics";
import { PasswordWall } from "./pages/PasswordWall";

function App() {
  return (
    <Router>
      <div className="min-h-screen bg-background relative overflow-hidden">
        {/* Subtle background glow effect */}
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-primary/10 rounded-full blur-[120px] pointer-events-none" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-secondary/10 rounded-full blur-[120px] pointer-events-none" />
        
        <Navbar />
        <main className="container mx-auto px-4 pt-24 pb-12 relative z-10">
          <Routes>
            <Route path="/" element={<Landing />} />
            <Route path="/login" element={<AuthPage type="login" />} />
            <Route path="/register" element={<AuthPage type="register" />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/analytics/:alias" element={<Analytics />} />
            <Route path="/p/:alias" element={<PasswordWall />} />
          </Routes>
        </main>
      </div>
    </Router>
  );
}

export default App;
