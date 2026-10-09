import { useState, useEffect } from "react";
import { Link, useNavigate } from "react-router-dom";
import { motion, AnimatePresence } from "framer-motion";
import { QRCodeSVG } from "qrcode.react";
import { GlassCard } from "../components/ui/GlassCard";
import { Button } from "../components/ui/Button";
import { Input } from "../components/ui/Input";
import { Link as LinkIcon, Settings, BarChart, Lock, Copy, Check, QrCode, Trash2, Power, Download } from "lucide-react";

export function Dashboard() {
  const navigate = useNavigate();
  const [links, setLinks] = useState([]);
  const [originalUrl, setOriginalUrl] = useState("");
  const [customAlias, setCustomAlias] = useState("");
  const [password, setPassword] = useState("");
  const [maxAccesses, setMaxAccesses] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [loading, setLoading] = useState(false);
  const [copiedAlias, setCopiedAlias] = useState(null);
  
  // QR Modal State
  const [qrModalUrl, setQrModalUrl] = useState(null);
  const [qrModalAlias, setQrModalAlias] = useState(null);

  useEffect(() => {
    if (!localStorage.getItem("token")) {
      navigate("/login");
      return;
    }
    fetchLinks();
  }, [navigate]);

  const fetchLinks = async () => {
    try {
      const res = await fetch("/api/v1/urls", {
        headers: { Authorization: `Bearer ${localStorage.getItem("token")}` }
      });
      if (res.ok) {
        const data = await res.json();
        setLinks(Array.isArray(data) ? data : []);
      } else if (res.status === 401) {
        localStorage.removeItem("token");
        navigate("/login");
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleShorten = async (e) => {
    e.preventDefault();
    setLoading(true);
    try {
      const payload = { original_url: originalUrl };
      if (customAlias) payload.custom_alias = customAlias;
      if (password) payload.password = password;
      if (maxAccesses) payload.max_accesses = parseInt(maxAccesses);

      const res = await fetch("/api/v1/urls", {
        method: "POST",
        headers: { 
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("token")}`
        },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        setOriginalUrl("");
        setCustomAlias("");
        setPassword("");
        setMaxAccesses("");
        setShowAdvanced(false);
        fetchLinks();
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const toggleEnable = async (shortCode, currentState) => {
    try {
      const res = await fetch(`/api/v1/urls/${shortCode}/enable`, {
        method: 'PUT',
        headers: { 
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem("token")}`
        },
        body: JSON.stringify({ is_enabled: !currentState })
      });
      if (res.ok) {
        fetchLinks();
      }
    } catch (err) {
      console.error(err);
    }
  };

  const deleteUrl = async (shortCode) => {
    if (!window.confirm('Are you sure you want to delete this link?')) return;
    try {
      const res = await fetch(`/api/v1/urls/${shortCode}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${localStorage.getItem("token")}` }
      });
      if (res.ok) {
        fetchLinks();
      }
    } catch (err) {
      console.error(err);
    }
  };

  const copyToClipboard = (alias) => {
    const url = `${window.location.origin}/${alias}`;
    navigator.clipboard.writeText(url);
    setCopiedAlias(alias);
    setTimeout(() => setCopiedAlias(null), 2000);
  };

  const downloadQR = () => {
    const svg = document.getElementById("qr-modal-svg");
    const svgData = new XMLSerializer().serializeToString(svg);
    const canvas = document.createElement("canvas");
    const ctx = canvas.getContext("2d");
    const img = new Image();
    img.onload = () => {
      canvas.width = img.width;
      canvas.height = img.height;
      ctx.drawImage(img, 0, 0);
      const pngFile = canvas.toDataURL("image/png");
      const downloadLink = document.createElement("a");
      downloadLink.download = `shorter-${qrModalAlias}.png`;
      downloadLink.href = `${pngFile}`;
      downloadLink.click();
    };
    img.src = "data:image/svg+xml;base64," + btoa(unescape(encodeURIComponent(svgData)));
  };

  return (
    <div className="space-y-12">
      {/* Shortener Section */}
      <section>
        <GlassCard className="max-w-4xl mx-auto">
          <h2 className="text-2xl font-bold text-textMain mb-6">Shorten a new link</h2>
          <form onSubmit={handleShorten} className="space-y-4">
            <div className="flex flex-col md:flex-row gap-4">
              <div className="flex-1">
                <Input 
                  placeholder="Enter the URL you want to shorten..."
                  icon={LinkIcon}
                  value={originalUrl}
                  onChange={(e) => setOriginalUrl(e.target.value)}
                  required
                />
              </div>
              <Button type="submit" variant="primary" isLoading={loading}>
                Shorten
              </Button>
            </div>
            
            <div>
              <button 
                type="button" 
                onClick={() => setShowAdvanced(!showAdvanced)}
                className="text-sm text-primary hover:underline flex items-center gap-1"
              >
                <Settings size={14} /> {showAdvanced ? "Hide advanced options" : "Show advanced options"}
              </button>
            </div>

            {showAdvanced && (
              <motion.div 
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: "auto" }}
                className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-4 border-t border-surfaceHighlight"
              >
                <Input 
                  placeholder="Custom alias" 
                  value={customAlias}
                  onChange={(e) => setCustomAlias(e.target.value)}
                />
                <Input 
                  type="password"
                  placeholder="Password protect" 
                  icon={Lock}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
                <Input 
                  type="number"
                  placeholder="Max accesses" 
                  value={maxAccesses}
                  onChange={(e) => setMaxAccesses(e.target.value)}
                  min="1"
                />
              </motion.div>
            )}
          </form>
        </GlassCard>
      </section>

      {/* Links List Section */}
      <section className="max-w-5xl mx-auto">
        <h2 className="text-2xl font-bold text-textMain mb-6">Your Links</h2>
        <div className="grid gap-4">
          {links.length === 0 ? (
            <div className="text-center text-textMuted py-12 glass-panel rounded-xl">
              No links created yet.
            </div>
          ) : (
            links.map((link, i) => (
              <motion.div 
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: i * 0.05 }}
                key={link.id}
                className={`glass-panel p-4 flex flex-col md:flex-row items-start md:items-center justify-between gap-4 group transition-opacity ${link.is_enabled === false ? 'opacity-50' : ''}`}
              >
                <div className="flex-1 min-w-0 pr-4">
                  <div className="flex items-center gap-2 mb-1">
                    <a href={`${window.location.origin}/${link.short_code}`} target="_blank" rel="noreferrer" className={`text-lg font-semibold hover:underline ${link.is_enabled === false ? 'text-textMuted' : 'text-primary'}`}>
                      /{link.short_code}
                    </a>
                    {link.password_hash && <Lock size={14} className="text-yellow-500" title="Password Protected" />}
                    {link.is_enabled === false && <span className="text-xs bg-red-500/20 text-red-400 px-2 py-0.5 rounded-full border border-red-500/30">Disabled</span>}
                  </div>
                  <p className="text-textMuted text-sm truncate" title={link.original_url}>
                    {link.original_url}
                  </p>
                  <div className="flex gap-4 mt-2 text-xs text-textMuted">
                    <span>Clicks: <span className="text-textMain">{link.access_count}</span> {link.max_accesses > 0 && `/ ${link.max_accesses}`}</span>
                    <span>Created: {new Date(link.created_at).toLocaleDateString()}</span>
                  </div>
                </div>

                <div className="flex items-center gap-1 flex-wrap justify-end">
                  <Button 
                    variant="ghost" 
                    className="p-2 h-auto" 
                    onClick={() => {
                      setQrModalUrl(`${window.location.origin}/${link.short_code}`);
                      setQrModalAlias(link.short_code);
                    }}
                    title="QR Code"
                  >
                    <QrCode size={18} />
                  </Button>
                  <Button 
                    variant="ghost" 
                    className="p-2 h-auto" 
                    onClick={() => copyToClipboard(link.short_code)}
                    title="Copy Link"
                  >
                    {copiedAlias === link.short_code ? <Check size={18} className="text-green-400" /> : <Copy size={18} />}
                  </Button>
                  <Button 
                    variant="ghost" 
                    className={`p-2 h-auto ${link.is_enabled === false ? 'text-green-400' : 'text-yellow-400'}`}
                    onClick={() => toggleEnable(link.short_code, link.is_enabled !== false)}
                    title={link.is_enabled === false ? "Enable Link" : "Disable Link"}
                  >
                    <Power size={18} />
                  </Button>
                  <Link to={`/analytics/${link.short_code}`}>
                    <Button variant="ghost" className="p-2 h-auto text-primary" title="Analytics">
                      <BarChart size={18} />
                    </Button>
                  </Link>
                  <Button 
                    variant="ghost" 
                    className="p-2 h-auto text-red-400 hover:text-red-300 hover:bg-red-400/10" 
                    onClick={() => deleteUrl(link.short_code)}
                    title="Delete"
                  >
                    <Trash2 size={18} />
                  </Button>
                </div>
              </motion.div>
            ))
          )}
        </div>
      </section>

      {/* QR Code Modal Overlay */}
      <AnimatePresence>
        {qrModalUrl && (
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
            onClick={() => setQrModalUrl(null)}
          >
            <motion.div 
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              onClick={(e) => e.stopPropagation()}
              className="bg-surface border border-surfaceHighlight rounded-2xl p-8 max-w-sm w-full shadow-2xl flex flex-col items-center text-center"
            >
              <h3 className="text-xl font-bold text-textMain mb-2">QR Code</h3>
              <p className="text-textMuted text-sm mb-6 truncate w-full">/{qrModalAlias}</p>
              
              <div className="bg-white p-4 rounded-xl mb-6">
                <QRCodeSVG 
                  id="qr-modal-svg"
                  value={qrModalUrl} 
                  size={200} 
                  bgColor={"#ffffff"}
                  fgColor={"#0a192f"}
                  level={"M"}
                />
              </div>

              <div className="flex gap-4 w-full">
                <Button variant="outline" className="flex-1" onClick={() => setQrModalUrl(null)}>
                  Close
                </Button>
                <Button variant="primary" className="flex-1 flex items-center justify-center gap-2" onClick={downloadQR}>
                  <Download size={16} /> Save
                </Button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
