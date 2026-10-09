import { motion } from "framer-motion";
import { Link } from "react-router-dom";
import { Button } from "../components/ui/Button";
import { Zap, Shield, BarChart3, Globe, Lock } from "lucide-react";
import { Meteors } from "../components/ui/Meteors";
import { cn } from "../lib/utils";

export function Landing() {
  const container = {
    hidden: { opacity: 0 },
    show: {
      opacity: 1,
      transition: { staggerChildren: 0.2 }
    }
  };

  const item = {
    hidden: { opacity: 0, y: 20 },
    show: { opacity: 1, y: 0, transition: { type: "spring", stiffness: 300, damping: 24 } }
  };

  return (
    <div className="flex flex-col items-center justify-center min-h-[80vh] text-center px-4 relative overflow-hidden">
      
      {/* Meteor Background Effect */}
      <div className="absolute inset-0 w-full h-full -z-10 overflow-hidden pointer-events-none">
        <Meteors number={15} />
      </div>

      <motion.div
        variants={container}
        initial="hidden"
        animate="show"
        className="max-w-4xl space-y-8 relative z-10 pt-20"
      >
        <motion.div variants={item} className="inline-block rounded-full bg-surfaceHighlight/50 border border-surfaceHighlight px-4 py-1.5 text-sm font-medium text-textMuted mb-4">
          Link Management Platform
        </motion.div>
        
        <motion.h1 variants={item} className="text-5xl md:text-7xl font-bold tracking-tight text-textMain leading-tight">
          Shorten links.<br/>
          <span className="text-primary">
            Expand your reach.
          </span>
        </motion.h1>
        
        <motion.p variants={item} className="text-xl text-textMuted max-w-2xl mx-auto">
          A programmable URL shortener built for developers and teams. Track clicks, protect with passwords, and manage access limits.
        </motion.p>
        
        <motion.div variants={item} className="flex flex-col sm:flex-row items-center justify-center gap-4 pt-4 pb-20">
          {localStorage.getItem("token") ? (
            <Link to="/dashboard" className="w-full sm:w-auto">
              <Button variant="primary" className="w-full sm:w-auto text-lg px-8 py-4 shadow-[0_0_30px_rgba(100,255,218,0.25)]">Go to Dashboard</Button>
            </Link>
          ) : (
            <>
              <Link to="/register" className="w-full sm:w-auto">
                <Button variant="primary" className="w-full sm:w-auto text-lg px-8 py-4 shadow-[0_0_30px_rgba(100,255,218,0.25)]">Start for free</Button>
              </Link>
              <Link to="/login" className="w-full sm:w-auto">
                <Button variant="secondary" className="w-full sm:w-auto text-lg px-8 py-4">Login to Dashboard</Button>
              </Link>
            </>
          )}
        </motion.div>

        {/* Animated Bento Grid */}
        <motion.div variants={item} className="grid grid-cols-1 md:grid-cols-3 md:grid-rows-2 gap-4 text-left pb-20 max-w-5xl mx-auto">
          <BentoCard 
            className="md:col-span-2 md:row-span-1"
            icon={<Zap className="text-primary w-8 h-8" />}
            title="Fast Edge Delivery"
            desc="Built on Go and Redis, our redirects are optimized for low latency and high concurrency."
          />
            <BentoCard 
            className="md:col-span-1 md:row-span-2 flex flex-col"
            icon={<BarChart3 className="text-secondary w-8 h-8" />}
            title="Analytics"
            desc="Track devices, browsers, and geographic locations to make data driven decisions."
            extra={<Globe className="w-full h-auto text-surfaceHighlight mt-auto opacity-50" />}
          />
          <BentoCard 
            className="md:col-span-1 md:row-span-1"
            icon={<Lock className="text-yellow-400 w-8 h-8" />}
            title="Password Protection"
            desc="Secure sensitive links with bcrypt hashed passwords."
          />
          <BentoCard 
            className="md:col-span-1 md:row-span-1"
            icon={<Shield className="text-purple-400 w-8 h-8" />}
            title="Access Limits"
            desc="Set maximum click limits or expiration dates easily."
          />
        </motion.div>
      </motion.div>
    </div>
  );
}

function BentoCard({ icon, title, desc, className, extra }) {
  return (
    <div 
      className={cn(
        "glass-panel p-8 rounded-3xl relative group overflow-hidden border border-surfaceHighlight/50 hover:border-surfaceHighlight transition-colors duration-300 bg-surface/50",
        className
      )}
    >
      <div className="relative z-10">
        <div className="w-14 h-14 rounded-2xl bg-surfaceHighlight/50 flex items-center justify-center mb-6 shadow-sm border border-white/5">
          {icon}
        </div>
        <h3 className="text-2xl font-semibold text-textMain mb-3 tracking-tight">{title}</h3>
        <p className="text-textMuted leading-relaxed">{desc}</p>
      </div>
      {extra && (
        <div className="relative z-0 mt-6 overflow-hidden">
          {extra}
        </div>
      )}
    </div>
  );
}
