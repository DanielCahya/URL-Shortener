import { motion } from "framer-motion";
import { cn } from "../../lib/utils";
import React from "react";

export const Button = React.forwardRef(({ className, variant = "primary", isLoading, children, ...props }, ref) => {
  const baseClasses = "inline-flex items-center justify-center font-semibold px-6 py-3 rounded-lg transition-all focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50 disabled:cursor-not-allowed";
  
  const variants = {
    primary: "bg-primary text-background hover:bg-primary/90 shadow-[0_0_15px_rgba(100,255,218,0.2)] hover:shadow-[0_0_25px_rgba(100,255,218,0.4)]",
    secondary: "bg-surfaceHighlight text-textMain hover:bg-surfaceHighlight/80 border border-surfaceHighlight/50",
    outline: "bg-transparent text-primary border border-primary hover:bg-primary/10",
    ghost: "bg-transparent text-textMain hover:bg-surfaceHighlight",
  };

  return (
    <motion.button
      ref={ref}
      whileHover={{ scale: 1.02 }}
      whileTap={{ scale: 0.98 }}
      className={cn(baseClasses, variants[variant], className)}
      disabled={isLoading || props.disabled}
      {...props}
    >
      {isLoading ? (
        <span className="flex items-center gap-2">
          <svg className="animate-spin h-5 w-5 text-current" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          Processing...
        </span>
      ) : (
        children
      )}
    </motion.button>
  );
});
Button.displayName = "Button";
