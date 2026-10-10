import { useState, useEffect } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { GlassCard } from "../components/ui/GlassCard";
import { Button } from "../components/ui/Button";
import { ArrowLeft, MousePointerClick, Globe, Monitor, Compass, MapPin } from "lucide-react";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip as RechartsTooltip } from "recharts";
import { motion } from "framer-motion";

const COLORS = ['#64ffda', '#0070f3', '#8b5cf6', '#ec4899', '#10b981', '#f59e0b'];

export function Analytics() {
  const { alias } = useParams();
  const navigate = useNavigate();
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!localStorage.getItem("token")) {
      navigate("/login");
      return;
    }
    
    const fetchAnalytics = async () => {
      try {
        const res = await fetch(`/api/v1/urls/${alias}/analytics`, {
          headers: { Authorization: `Bearer ${localStorage.getItem("token")}` }
        });
        if (res.ok) {
          const json = await res.json();
          setData(json);
        } else if (res.status === 401) {
          localStorage.removeItem("token");
          navigate("/login");
        }
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    
    fetchAnalytics();
  }, [alias, navigate]);


  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[50vh] space-y-4">
        <div className="w-12 h-12 border-4 border-primary/20 border-t-primary rounded-full animate-spin"></div>
        <p className="text-textMuted animate-pulse">Loading analytics...</p>
      </div>
    );
  }

  if (!data) {
    return <div className="text-center py-20 text-red-400">Analytics data not found.</div>;
  }

  const { short_url, total_clicks } = data;

  // Find top country safely
  let topCountry = "-";
  let countryData = [];
  if (data.by_country) {
    countryData = Object.entries(data.by_country)
      .map(([code, count]) => ({ code: code || 'Unknown', count }))
      .sort((a, b) => b.count - a.count);
    
    if (countryData.length > 0) {
      topCountry = countryData[0].code;
    }
  }

  const formatChartData = (obj) => {
    if (!obj) return [];
    return Object.entries(obj)
      .map(([name, value]) => ({ name: name || 'Unknown', value }))
      .sort((a, b) => b.value - a.value);
  };

  const deviceData = formatChartData(data.by_device);
  const browserData = formatChartData(data.by_browser);

  return (
    <div className="max-w-5xl mx-auto space-y-8 px-4">
      <Link to="/dashboard" className="inline-flex items-center text-textMuted hover:text-primary transition-colors">
        <ArrowLeft size={16} className="mr-2" /> Back to Dashboard
      </Link>
      
      <div className="flex flex-col md:flex-row justify-between items-start gap-4">
        <motion.div initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }}>
          <h1 className="text-3xl font-bold text-textMain tracking-tight">Analytics Overview</h1>
          <p className="text-textMuted text-lg mt-1">/{alias}</p>
        </motion.div>
      </div>

      <GlassCard className="p-8 border-surfaceHighlight/50">
        {/* Top Stats */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-12">
          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            className="bg-surfaceHighlight/30 p-6 rounded-2xl border border-surfaceHighlight/50 relative overflow-hidden group"
          >
            <div className="absolute inset-0 bg-primary/5 opacity-0 group-hover:opacity-100 transition-opacity" />
            <div className="flex items-center gap-3 text-textMuted mb-2">
              <MousePointerClick size={20} className="text-primary" />
              <h3 className="font-medium">Total Clicks</h3>
            </div>
            <p className="text-5xl font-bold text-textMain">{total_clicks}</p>
          </motion.div>

          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="bg-surfaceHighlight/30 p-6 rounded-2xl border border-surfaceHighlight/50 relative overflow-hidden group"
          >
            <div className="absolute inset-0 bg-secondary/5 opacity-0 group-hover:opacity-100 transition-opacity" />
            <div className="flex items-center gap-3 text-textMuted mb-2">
              <Globe size={20} className="text-secondary" />
              <h3 className="font-medium">Top Country</h3>
            </div>
            <p className="text-5xl font-bold text-textMain">{topCountry}</p>
          </motion.div>
        </div>
        
        {/* Charts & Details */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          
          {/* Device Chart */}
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.2 }} className="col-span-1">
            <div className="flex items-center gap-2 text-textMuted mb-6 border-b border-surfaceHighlight/50 pb-4">
              <Monitor size={18} />
              <h4 className="font-medium text-lg text-textMain">Devices</h4>
            </div>
            
            {deviceData.length > 0 ? (
              <div className="flex flex-col items-center">
                <div className="h-[220px] w-full relative">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={deviceData}
                        cx="50%"
                        cy="50%"
                        innerRadius={60}
                        outerRadius={80}
                        paddingAngle={5}
                        dataKey="value"
                      >
                        {deviceData.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                        ))}
                      </Pie>
                      <RechartsTooltip 
                        contentStyle={{ backgroundColor: '#0a192f', borderColor: '#112240', borderRadius: '8px', color: '#ccd6f6' }}
                        itemStyle={{ color: '#64ffda' }}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
                {/* Legend */}
                <div className="flex flex-wrap justify-center gap-4 mt-6 w-full">
                  {deviceData.map((entry, index) => (
                    <div key={entry.name} className="flex items-center gap-2 text-sm text-textMuted">
                      <span className="w-3 h-3 rounded-full" style={{ backgroundColor: COLORS[index % COLORS.length] }}></span>
                      {entry.name} <span className="font-medium text-textMain">({entry.value})</span>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="h-[200px] flex items-center justify-center text-textMuted bg-surfaceHighlight/10 rounded-xl border border-surfaceHighlight/30 dashed">
                No device data yet
              </div>
            )}
          </motion.div>
          
          {/* Browser Chart */}
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.3 }} className="col-span-1">
            <div className="flex items-center gap-2 text-textMuted mb-6 border-b border-surfaceHighlight/50 pb-4">
              <Compass size={18} />
              <h4 className="font-medium text-lg text-textMain">Browsers</h4>
            </div>
            
            {browserData.length > 0 ? (
              <div className="flex flex-col items-center">
                <div className="h-[220px] w-full relative">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={browserData}
                        cx="50%"
                        cy="50%"
                        innerRadius={60}
                        outerRadius={80}
                        paddingAngle={5}
                        dataKey="value"
                      >
                        {browserData.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={COLORS[(index + 2) % COLORS.length]} />
                        ))}
                      </Pie>
                      <RechartsTooltip 
                        contentStyle={{ backgroundColor: '#0a192f', borderColor: '#112240', borderRadius: '8px', color: '#ccd6f6' }}
                        itemStyle={{ color: '#0070f3' }}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
                {/* Legend */}
                <div className="flex flex-wrap justify-center gap-4 mt-6 w-full">
                  {browserData.map((entry, index) => (
                    <div key={entry.name} className="flex items-center gap-2 text-sm text-textMuted">
                      <span className="w-3 h-3 rounded-full" style={{ backgroundColor: COLORS[(index + 2) % COLORS.length] }}></span>
                      {entry.name} <span className="font-medium text-textMain">({entry.value})</span>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              <div className="h-[200px] flex items-center justify-center text-textMuted bg-surfaceHighlight/10 rounded-xl border border-surfaceHighlight/30 dashed">
                No browser data yet
              </div>
            )}
          </motion.div>

          {/* Countries List */}
          <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ delay: 0.4 }} className="col-span-1">
            <div className="flex items-center gap-2 text-textMuted mb-6 border-b border-surfaceHighlight/50 pb-4">
              <MapPin size={18} />
              <h4 className="font-medium text-lg text-textMain">All Countries</h4>
            </div>
            
            {countryData.length > 0 ? (
              <div className="space-y-3 max-h-[300px] overflow-y-auto pr-2 custom-scrollbar">
                {countryData.map((country, index) => (
                  <div key={index} className="flex items-center justify-between p-3 rounded-lg bg-surfaceHighlight/20 border border-surfaceHighlight/30 hover:bg-surfaceHighlight/40 transition-colors">
                    <span className="text-textMain font-medium flex items-center gap-2">
                      <span className="text-xl">{country.code === 'Unknown' ? '🌍' : `🇺🇳`}</span> 
                      {country.code}
                    </span>
                    <span className="text-primary font-bold bg-primary/10 px-3 py-1 rounded-full text-sm">
                      {country.count} clicks
                    </span>
                  </div>
                ))}
              </div>
            ) : (
              <div className="h-[200px] flex items-center justify-center text-textMuted bg-surfaceHighlight/10 rounded-xl border border-surfaceHighlight/30 dashed">
                No location data yet
              </div>
            )}
          </motion.div>

        </div>
      </GlassCard>
    </div>
  );
}
