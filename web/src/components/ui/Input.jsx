import React from "react";
import { cn } from "../../lib/utils";
import { motion, AnimatePresence } from "framer-motion";

export const Input = React.forwardRef(({ className, label, error, icon: Icon, ...props }, ref) => {
  return (
    <div className="w-full flex flex-col gap-1.5">
      {label && (
        <label className="text-sm font-medium text-textMain ml-1">
          {label}
        </label>
      )}
      <div className="relative flex items-center">
        {Icon && (
          <div className="absolute left-3 text-textMuted">
            <Icon size={18} />
          </div>
        )}
        <input
          ref={ref}
          className={cn(
            "input-field",
            Icon && "pl-10",
            error && "border-red-500/50 focus:border-red-500 focus:ring-red-500/20",
            className
          )}
          {...props}
        />
      </div>
      <AnimatePresence>
        {error && (
          <motion.span
            initial={{ opacity: 0, y: -5 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -5 }}
            className="text-xs text-red-400 ml-1"
          >
            {error}
          </motion.span>
        )}
      </AnimatePresence>
    </div>
  );
});
Input.displayName = "Input";
