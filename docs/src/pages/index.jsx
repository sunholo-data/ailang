import React from 'react';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import {
  Zap,
  Shield,
  Code2,
  Bot,
  Layers,
  GitBranch,
  ArrowRight,
  Terminal,
  Sparkles,
  Cpu,
  BookOpen,
  Play,
  RefreshCw,
  Activity,
  Lock,
  Copy,
  Check,
  Globe,
  Download
} from 'lucide-react';
import { STABLE_RELEASE, ACTIVE_PROMPT } from '../constants/version';
import BenchmarkMini from '../components/BenchmarkMini';
import AddToAgent from '../components/AddToAgent';

// Hero Section
function HeroSection() {
  return (
    <section className="hero-section">
      <div className="hero-bg">
        <div className="hero-gradient-orb hero-gradient-orb-1" />
        <div className="hero-gradient-orb hero-gradient-orb-2" />
        <div className="hero-grid" />
      </div>

      <div className="hero-content">
        <div className="hero-logo">
          <img src="/img/ailang-logo.svg" alt="AILANG" width="120" height="120" />
        </div>

        <div className="hero-badge">
          <Sparkles size={14} />
          <span>Version {STABLE_RELEASE} Released</span>
        </div>

        <h1 className="hero-title">
          The <span className="hero-title-gradient">AI-First</span>
          <br />
          Programming Language
        </h1>

        <p className="hero-subtitle">
          AILANG makes AI-generated code cheaper to debug, replay, and fix.
          Explicit effects constrain what code can do. Structured traces
          make errors easy to localize.
        </p>

        <div className="hero-actions">
          <Link to="/docs/guides/getting-started" className="hero-btn hero-btn-primary">
            <Play size={18} />
            Get Started
          </Link>
          <Link to="/docs" className="hero-btn hero-btn-secondary">
            <BookOpen size={18} />
            Documentation
          </Link>
          <a href="https://www.sunholo.com/ailang-demos/" className="hero-btn hero-btn-accent" target="_blank" rel="noopener noreferrer">
            <Globe size={18} />
            Live Demos
          </a>
        </div>

        <div className="hero-code">
          <div className="hero-code-header">
            <span className="hero-code-dot red" />
            <span className="hero-code-dot yellow" />
            <span className="hero-code-dot green" />
            <span className="hero-code-filename">hello.ail</span>
          </div>
          <pre className="hero-code-content">
{`-- hello.ail
module examples/hello

export func main() -> () ! {IO} =
  print("Hello, AILANG!")`}
          </pre>
        </div>
      </div>

      <style>{`
        .hero-section {
          position: relative;
          min-height: 90vh;
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 4rem 2rem;
          overflow: hidden;
          background: linear-gradient(135deg, #0a0e14 0%, #0f1419 50%, #131a24 100%);
        }

        .hero-bg {
          position: absolute;
          inset: 0;
          overflow: hidden;
        }

        .hero-gradient-orb {
          position: absolute;
          border-radius: 50%;
          filter: blur(100px);
          opacity: 0.4;
        }

        .hero-gradient-orb-1 {
          width: 600px;
          height: 600px;
          background: radial-gradient(circle, #e73c17 0%, transparent 70%);
          top: -200px;
          right: -100px;
          animation: float 20s ease-in-out infinite;
        }

        .hero-gradient-orb-2 {
          width: 400px;
          height: 400px;
          background: radial-gradient(circle, #2c7a7b 0%, transparent 70%);
          bottom: -100px;
          left: -50px;
          animation: float 15s ease-in-out infinite reverse;
        }

        .hero-grid {
          position: absolute;
          inset: 0;
          background-image:
            linear-gradient(rgba(255,255,255,0.02) 1px, transparent 1px),
            linear-gradient(90deg, rgba(255,255,255,0.02) 1px, transparent 1px);
          background-size: 50px 50px;
          mask-image: radial-gradient(ellipse at center, black 20%, transparent 70%);
        }

        @keyframes float {
          0%, 100% { transform: translate(0, 0) scale(1); }
          33% { transform: translate(30px, -30px) scale(1.05); }
          66% { transform: translate(-20px, 20px) scale(0.95); }
        }

        .hero-content {
          position: relative;
          z-index: 1;
          max-width: 900px;
          text-align: center;
        }

        .hero-logo {
          margin-bottom: 1.5rem;
          animation: fadeInUp 0.5s ease-out forwards;
          opacity: 0;
        }

        .hero-logo img {
          filter: drop-shadow(0 0 30px rgba(231, 60, 23, 0.4));
          transition: filter 0.3s ease;
        }

        .hero-logo img:hover {
          filter: drop-shadow(0 0 50px rgba(231, 60, 23, 0.6));
        }

        .hero-badge {
          display: inline-flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.5rem 1rem;
          background: rgba(231, 60, 23, 0.1);
          border: 1px solid rgba(231, 60, 23, 0.3);
          border-radius: 50px;
          font-family: 'Montserrat', sans-serif;
          font-weight: 600;
          font-size: 0.85rem;
          color: #ff5a3c;
          margin-bottom: 2rem;
          animation: fadeInUp 0.6s ease-out forwards;
        }

        .hero-title {
          font-family: 'Montserrat', sans-serif;
          font-weight: 800;
          font-size: clamp(2.5rem, 8vw, 4.5rem);
          line-height: 1.1;
          color: white;
          margin-bottom: 1.5rem;
          animation: fadeInUp 0.6s ease-out 0.1s forwards;
          opacity: 0;
        }

        .hero-title-gradient {
          background: linear-gradient(135deg, #e73c17, #ff5a3c, #dd6b20);
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        }

        .hero-subtitle {
          font-size: clamp(1rem, 2.5vw, 1.25rem);
          color: rgba(255, 255, 255, 0.7);
          max-width: 600px;
          margin: 0 auto 2.5rem;
          line-height: 1.7;
          animation: fadeInUp 0.6s ease-out 0.2s forwards;
          opacity: 0;
        }

        .hero-actions {
          display: flex;
          gap: 1rem;
          justify-content: center;
          flex-wrap: wrap;
          margin-bottom: 3rem;
          animation: fadeInUp 0.6s ease-out 0.3s forwards;
          opacity: 0;
        }

        .hero-btn {
          display: inline-flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.875rem 1.75rem;
          font-family: 'Montserrat', sans-serif;
          font-weight: 600;
          font-size: 1rem;
          border-radius: 10px;
          transition: all 0.3s ease;
          text-decoration: none;
        }

        .hero-btn-primary {
          background: linear-gradient(135deg, #e73c17, #dd6b20);
          color: white;
          box-shadow: 0 4px 20px rgba(231, 60, 23, 0.4);
        }

        .hero-btn-primary:hover {
          transform: translateY(-3px);
          box-shadow: 0 8px 30px rgba(231, 60, 23, 0.5);
          color: white;
        }

        .hero-btn-secondary {
          background: rgba(255, 255, 255, 0.05);
          border: 1px solid rgba(255, 255, 255, 0.2);
          color: white;
        }

        .hero-btn-secondary:hover {
          background: rgba(255, 255, 255, 0.1);
          border-color: rgba(231, 60, 23, 0.5);
          color: white;
        }

        .hero-btn-accent {
          background: rgba(44, 122, 123, 0.15);
          border: 1px solid rgba(44, 122, 123, 0.4);
          color: #4ecdc4;
        }

        .hero-btn-accent:hover {
          background: rgba(44, 122, 123, 0.25);
          border-color: rgba(44, 122, 123, 0.7);
          transform: translateY(-3px);
          box-shadow: 0 8px 30px rgba(44, 122, 123, 0.3);
          color: #4ecdc4;
        }

        .hero-code {
          background: #1a2332;
          border-radius: 16px;
          overflow: hidden;
          text-align: left;
          box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
          border: 1px solid rgba(255, 255, 255, 0.05);
          animation: fadeInUp 0.6s ease-out 0.4s forwards;
          opacity: 0;
          max-width: 500px;
          margin: 0 auto;
        }

        .hero-code-header {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.75rem 1rem;
          background: rgba(0, 0, 0, 0.3);
          border-bottom: 1px solid rgba(255, 255, 255, 0.05);
        }

        .hero-code-dot {
          width: 12px;
          height: 12px;
          border-radius: 50%;
        }

        .hero-code-dot.red { background: #ff5f56; }
        .hero-code-dot.yellow { background: #ffbd2e; }
        .hero-code-dot.green { background: #27ca40; }

        .hero-code-filename {
          margin-left: auto;
          font-family: 'JetBrains Mono', monospace;
          font-size: 0.75rem;
          color: rgba(255, 255, 255, 0.5);
        }

        .hero-code-content {
          margin: 0;
          padding: 1.5rem;
          font-family: 'JetBrains Mono', monospace;
          font-size: 0.9rem;
          line-height: 1.7;
          color: #e2e8f0;
          background: transparent;
          overflow-x: auto;
        }

        @keyframes fadeInUp {
          from {
            opacity: 0;
            transform: translateY(20px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }
      `}</style>
    </section>
  );
}

// Quick Start Section
function QuickStartSection() {
  const [copiedInstall, setCopiedInstall] = React.useState(false);
  const [copiedClaude, setCopiedClaude] = React.useState(false);
  const [copiedCodex, setCopiedCodex] = React.useState(false);

  const copyToClipboard = (text, setCopied) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section className="quickstart-section">
      <div className="quickstart-container">
        <div className="quickstart-header">
          <h2 className="quickstart-title">Start in 30 Seconds</h2>
          <p className="quickstart-subtitle">
            One command to install, or use your AI coding agent's plugin system
          </p>
        </div>

        <div className="quickstart-install">
          <div className="quickstart-card quickstart-card-featured">
            <div className="quickstart-card-header">
              <Download size={24} />
              <span>Quick Install</span>
            </div>
            <div className="quickstart-code">
              <pre>curl -fsSL https://ailang.sunholo.com/install.sh | bash</pre>
              <button
                className="quickstart-copy"
                onClick={() => copyToClipboard('curl -fsSL https://ailang.sunholo.com/install.sh | bash', setCopiedInstall)}
              >
                {copiedInstall ? <Check size={16} /> : <Copy size={16} />}
              </button>
            </div>
          </div>
        </div>

        <div className="quickstart-grid">
          <div className="quickstart-card">
            <div className="quickstart-card-header">
              <Bot size={24} />
              <span>Claude Code</span>
            </div>
            <div className="quickstart-code">
              <pre>/plugin marketplace add sunholo-data/ailang_bootstrap{'\n'}/plugin install ailang</pre>
              <button
                className="quickstart-copy"
                onClick={() => copyToClipboard('/plugin marketplace add sunholo-data/ailang_bootstrap\n/plugin install ailang', setCopiedClaude)}
              >
                {copiedClaude ? <Check size={16} /> : <Copy size={16} />}
              </button>
            </div>
          </div>

          <div className="quickstart-card">
            <div className="quickstart-card-header">
              <Terminal size={24} />
              <span>Codex CLI</span>
            </div>
            <div className="quickstart-code">
              <pre>codex plugin add https://github.com/sunholo-data/ailang_bootstrap</pre>
              <button
                className="quickstart-copy"
                onClick={() => copyToClipboard('codex plugin add https://github.com/sunholo-data/ailang_bootstrap', setCopiedCodex)}
              >
                {copiedCodex ? <Check size={16} /> : <Copy size={16} />}
              </button>
            </div>
          </div>
        </div>

        <div className="quickstart-footer">
          <p>Then just ask your agent: <em>"Write an AILANG program that reads a file and counts lines"</em></p>
          <Link to="https://github.com/sunholo-data/ailang_bootstrap" className="quickstart-link">
            View ailang_bootstrap on GitHub <ArrowRight size={14} />
          </Link>
        </div>
      </div>

      <style>{`
        .quickstart-section {
          padding: 4rem 2rem;
          background: #0a0a0a !important;
        }

        .quickstart-container {
          max-width: 900px;
          margin: 0 auto;
        }

        .quickstart-header {
          text-align: center;
          margin-bottom: 2.5rem;
        }

        .quickstart-title {
          font-family: 'Montserrat', sans-serif;
          font-weight: 800;
          font-size: clamp(1.75rem, 4vw, 2.5rem);
          color: #ffffff !important;
          margin-bottom: 0.5rem;
        }

        .quickstart-subtitle {
          font-size: 1.1rem;
          color: rgba(255, 255, 255, 0.7) !important;
        }

        .quickstart-install {
          max-width: 600px;
          margin: 0 auto 1.5rem;
        }

        .quickstart-card-featured {
          border-color: rgba(231, 60, 23, 0.4) !important;
          box-shadow: 0 0 20px rgba(231, 60, 23, 0.1);
        }

        .quickstart-card-featured .quickstart-code pre {
          font-size: 0.95rem;
        }

        .quickstart-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 1.5rem;
          margin-bottom: 2rem;
        }

        .quickstart-card {
          background: #1a1a1a !important;
          border: 1px solid #333333 !important;
          border-radius: 12px;
          overflow: hidden;
        }

        .quickstart-card-header {
          display: flex;
          align-items: center;
          gap: 0.75rem;
          padding: 1rem 1.25rem;
          background: #111111 !important;
          border-bottom: 1px solid #333333 !important;
          color: #ffffff !important;
          font-family: 'Montserrat', sans-serif;
          font-weight: 600;
        }

        .quickstart-card-header svg {
          color: #ff5a3c !important;
        }

        .quickstart-code {
          position: relative;
          padding: 1.25rem;
          background: #1a1a1a !important;
        }

        .quickstart-code pre {
          margin: 0;
          font-family: 'JetBrains Mono', monospace;
          font-size: 0.85rem;
          line-height: 1.6;
          color: #ffffff !important;
          background: transparent !important;
          white-space: pre-wrap;
          word-break: break-all;
        }

        .quickstart-copy {
          position: absolute;
          top: 0.75rem;
          right: 0.75rem;
          background: #333333 !important;
          border: none;
          border-radius: 6px;
          padding: 0.5rem;
          cursor: pointer;
          color: rgba(255, 255, 255, 0.7) !important;
          transition: all 0.2s ease;
        }

        .quickstart-copy:hover {
          background: #444444 !important;
          color: #ffffff !important;
        }

        .quickstart-footer {
          text-align: center;
          color: rgba(255, 255, 255, 0.7) !important;
          font-size: 0.95rem;
        }

        .quickstart-footer em {
          color: rgba(255, 255, 255, 0.9) !important;
        }

        .quickstart-link {
          display: inline-flex;
          align-items: center;
          gap: 0.5rem;
          margin-top: 1rem;
          color: #ff5a3c !important;
          font-weight: 500;
          text-decoration: none;
          transition: gap 0.2s ease;
        }

        .quickstart-link:hover {
          gap: 0.75rem;
          color: #ff7a5c !important;
        }
      `}</style>
    </section>
  );
}

// Features Section - Aligned with AILANG Design Axioms
const features = [
  {
    icon: RefreshCw,
    title: 'Deterministic Execution',
    description: 'Same input, same output, every time. Replay any execution for debugging. No hidden nondeterminism.',
    color: '#e73c17',
    axiom: 'Axiom 1'
  },
  {
    icon: Shield,
    title: 'Effect Boundaries',
    description: 'Side effects are explicit in types. AI cannot hallucinate network calls in pure functions.',
    color: '#2c7a7b',
    axiom: 'Axiom 3 & 4'
  },
  {
    icon: Activity,
    title: 'Structured Traces',
    description: 'See exactly what happened. Slice traces by effect type. Get specific feedback, not "it crashed."',
    color: '#6b46c1',
    axiom: 'Axiom 2'
  },
  {
    icon: Cpu,
    title: 'Machine-First Design',
    description: 'Built for AI reasoning, not human ergonomics. Decidable structure and semantic compression.',
    color: '#2b6cb0',
    axiom: 'Axiom 7'
  },
  {
    icon: Lock,
    title: 'Explicit Authority',
    description: 'No implicit access to the world. Capabilities are statically visible and constrained by budget.',
    color: '#dd6b20',
    axiom: 'Axiom 4'
  },
  {
    icon: Zap,
    title: 'Pure Functional Core',
    description: 'Lambda calculus, pattern matching, ADTs. Composable features that never break reasoning.',
    color: '#38a169',
    axiom: 'Axiom 10'
  }
];

function FeaturesSection() {
  return (
    <section className="features-section">
      <div className="features-container">
        <div className="features-header">
          <h2 className="features-title">Built on 12 Design Axioms</h2>
          <p className="features-subtitle">
            Every feature derives from non-negotiable principles that make AI-generated code easier to debug, replay, and fix.
            <br />
            <Link to="/docs/references/axioms" className="features-axioms-link">
              Read the full axioms <ArrowRight size={14} />
            </Link>
          </p>
        </div>

        <div className="features-grid">
          {features.map((feature, index) => (
            <div
              key={feature.title}
              className="feature-card"
              style={{ animationDelay: `${index * 0.1}s` }}
            >
              <div
                className="feature-icon"
                style={{ background: `linear-gradient(135deg, ${feature.color}, ${feature.color}88)` }}
              >
                <feature.icon size={24} color="white" />
              </div>
              <h3 className="feature-title">{feature.title}</h3>
              <p className="feature-description">{feature.description}</p>
              {feature.axiom && (
                <span className="feature-axiom">{feature.axiom}</span>
              )}
            </div>
          ))}
        </div>
      </div>

      <style>{`
        .features-section {
          padding: 6rem 2rem;
          background: var(--ifm-background-color);
        }

        .features-container {
          max-width: 1200px;
          margin: 0 auto;
        }

        .features-header {
          text-align: center;
          margin-bottom: 4rem;
        }

        .features-title {
          font-family: 'Montserrat', sans-serif;
          font-weight: 800;
          font-size: clamp(2rem, 5vw, 3rem);
          margin-bottom: 1rem;
          background: linear-gradient(135deg, var(--ifm-font-color-base), var(--ifm-font-color-secondary));
          -webkit-background-clip: text;
          -webkit-text-fill-color: transparent;
          background-clip: text;
        }

        .features-subtitle {
          font-size: 1.15rem;
          color: var(--ifm-font-color-secondary);
          max-width: 700px;
          margin: 0 auto;
        }

        .features-axioms-link {
          display: inline-flex;
          align-items: center;
          gap: 0.4rem;
          margin-top: 0.75rem;
          color: var(--ifm-color-primary);
          font-weight: 500;
          font-size: 0.95rem;
          text-decoration: none;
          transition: gap 0.2s ease;
        }

        .features-axioms-link:hover {
          gap: 0.6rem;
        }

        .features-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
          gap: 2rem;
        }

        .feature-card {
          background: var(--ifm-background-surface-color);
          border-radius: 16px;
          padding: 2rem;
          border: 1px solid rgba(128, 128, 128, 0.1);
          transition: all 0.3s ease;
          animation: fadeInUp 0.6s ease-out forwards;
          opacity: 0;
        }

        .feature-card:hover {
          transform: translateY(-8px);
          box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
          border-color: var(--ifm-color-primary);
        }

        [data-theme='dark'] .feature-card:hover {
          box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
        }

        .feature-icon {
          width: 56px;
          height: 56px;
          border-radius: 14px;
          display: flex;
          align-items: center;
          justify-content: center;
          margin-bottom: 1.25rem;
        }

        .feature-title {
          font-family: 'Montserrat', sans-serif;
          font-weight: 700;
          font-size: 1.25rem;
          margin-bottom: 0.75rem;
        }

        .feature-description {
          color: var(--ifm-font-color-secondary);
          line-height: 1.7;
          margin: 0;
        }

        .feature-axiom {
          display: inline-block;
          margin-top: 1rem;
          padding: 0.25rem 0.75rem;
          background: rgba(231, 60, 23, 0.1);
          border: 1px solid rgba(231, 60, 23, 0.2);
          border-radius: 20px;
          font-size: 0.75rem;
          font-weight: 600;
          color: var(--ifm-color-primary);
        }

        @keyframes fadeInUp {
          from {
            opacity: 0;
            transform: translateY(20px);
          }
          to {
            opacity: 1;
            transform: translateY(0);
          }
        }
      `}</style>
    </section>
  );
}

// Demos Showcase Section
const demos = [
  {
    title: 'AI + Contracts',
    description: 'AI extracts structured data from documents. AILANG contracts validate every field before returning — deterministic verification of stochastic AI output.',
    url: 'https://www.sunholo.com/ailang-demos/contracts-ai.html',
    icon: Shield,
    color: '#2c7a7b',
    badges: ['WASM', 'Z3', 'AI Effect'],
  },
  {
    title: 'Safe Agent',
    description: 'Contract-verified AI tool calling. Calculator, file reader, and SQL tools wrapped in requires/ensures contracts — invalid arguments blocked before execution.',
    url: 'https://www.sunholo.com/ailang-demos/streaming/safe_agent/',
    icon: Lock,
    color: '#6b46c1',
    badges: ['Streaming', 'Contracts', 'Tools'],
  },
  {
    title: 'DocParse',
    description: 'Parse DOCX, PPTX, XLSX, and PDF entirely in-browser using WebAssembly. 10 AILANG modules, 28 contracts, zero server round-trips.',
    url: 'https://www.sunholo.com/ailang-demos/docparse.html',
    icon: Code2,
    color: '#dd6b20',
    badges: ['WASM', 'std/xml', 'std/zip'],
  },
  {
    title: 'Gemini Live',
    description: 'Bidirectional WebSocket audio streaming with 30 voice options. Real-time text-to-speech powered by AILANG effect handlers.',
    url: 'https://www.sunholo.com/ailang-demos/streaming/gemini_live/',
    icon: Activity,
    color: '#e73c17',
    badges: ['WebSocket', 'Audio', 'Streaming'],
  },
];

function DemosShowcase() {
  return (
    <section className="demos-section">
      <div className="demos-container">
        <div className="demos-header">
          <h2 className="demos-title">See It In Action</h2>
          <p className="demos-subtitle">
            Interactive browser demos — no installation required. All running on AILANG WebAssembly.
          </p>
        </div>

        <div className="demos-grid">
          {demos.map((demo, index) => (
            <a
              key={demo.title}
              href={demo.url}
              className="demo-card"
              target="_blank"
              rel="noopener noreferrer"
              style={{ animationDelay: `${index * 0.1}s` }}
            >
              <div
                className="demo-icon"
                style={{ background: `linear-gradient(135deg, ${demo.color}, ${demo.color}88)` }}
              >
                <demo.icon size={24} color="white" />
              </div>
              <h3 className="demo-card-title">{demo.title}</h3>
              <p className="demo-card-description">{demo.description}</p>
              <div className="demo-badges">
                {demo.badges.map(badge => (
                  <span key={badge} className="demo-badge">{badge}</span>
                ))}
              </div>
              <span className="demo-card-link">
                Try it <ArrowRight size={14} />
              </span>
            </a>
          ))}
        </div>

        <div className="demos-footer">
          <a href="https://www.sunholo.com/ailang-demos/" className="demos-view-all" target="_blank" rel="noopener noreferrer">
            View All 9 Demos <ArrowRight size={16} />
          </a>
        </div>
      </div>

      <style>{`
        .demos-section {
          padding: 6rem 2rem;
          background: linear-gradient(180deg, var(--ifm-background-color) 0%, var(--ifm-background-surface-color) 100%);
        }

        .demos-container {
          max-width: 1200px;
          margin: 0 auto;
        }

        .demos-header {
          text-align: center;
          margin-bottom: 4rem;
        }

        .demos-title {
          font-family: 'Montserrat', sans-serif;
          font-weight: 800;
          font-size: clamp(2rem, 5vw, 3rem);
          margin-bottom: 1rem;
          color: var(--ifm-heading-color);
        }

        .demos-subtitle {
          font-size: 1.15rem;
          color: var(--ifm-color-emphasis-700, rgba(255, 255, 255, 0.7));
          max-width: 600px;
          margin: 0 auto;
        }

        .demos-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
          gap: 1.5rem;
          margin-bottom: 3rem;
        }

        .demo-card {
          background: var(--ifm-background-surface-color);
          border-radius: 16px;
          padding: 2rem;
          border: 1px solid var(--ifm-color-emphasis-200);
          transition: all 0.3s ease;
          animation: fadeInUp 0.6s ease-out forwards;
          opacity: 0;
          text-decoration: none;
          color: inherit;
          display: flex;
          flex-direction: column;
          box-shadow: var(--card-shadow);
        }

        .demo-card:hover {
          transform: translateY(-8px);
          border-color: rgba(231, 60, 23, 0.4);
          box-shadow: var(--card-shadow-hover);
          background: var(--ifm-background-surface-color);
        }

        .demo-icon {
          width: 56px;
          height: 56px;
          border-radius: 14px;
          display: flex;
          align-items: center;
          justify-content: center;
          margin-bottom: 1.25rem;
        }

        .demo-card-title {
          font-family: 'Montserrat', sans-serif;
          font-weight: 700;
          font-size: 1.25rem;
          margin-bottom: 0.75rem;
          color: var(--ifm-heading-color);
        }

        .demo-card-description {
          color: var(--ifm-color-emphasis-700);
          line-height: 1.7;
          font-size: 0.95rem;
          margin: 0 0 1rem 0;
          flex-grow: 1;
        }

        .demo-badges {
          display: flex;
          flex-wrap: wrap;
          gap: 0.5rem;
          margin-bottom: 1.25rem;
        }

        .demo-badge {
          padding: 0.2rem 0.6rem;
          background: var(--ifm-color-emphasis-100);
          border: 1px solid var(--ifm-color-emphasis-200);
          border-radius: 20px;
          font-size: 0.7rem;
          font-weight: 600;
          color: var(--ifm-color-emphasis-600);
          font-family: 'JetBrains Mono', monospace;
        }

        .demo-card-link {
          display: inline-flex;
          align-items: center;
          gap: 0.4rem;
          font-weight: 600;
          font-size: 0.9rem;
          color: var(--ifm-color-primary);
          transition: gap 0.2s ease;
        }

        .demo-card:hover .demo-card-link {
          gap: 0.7rem;
        }

        .demos-footer {
          text-align: center;
        }

        .demos-view-all {
          display: inline-flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.875rem 2rem;
          background: rgba(231, 60, 23, 0.1);
          border: 1px solid rgba(231, 60, 23, 0.3);
          border-radius: 10px;
          color: var(--ifm-color-primary);
          font-family: 'Montserrat', sans-serif;
          font-weight: 600;
          font-size: 1rem;
          text-decoration: none;
          transition: all 0.3s ease;
        }

        .demos-view-all:hover {
          background: rgba(231, 60, 23, 0.2);
          border-color: rgba(231, 60, 23, 0.5);
          transform: translateY(-2px);
          color: var(--ifm-color-primary);
          gap: 0.75rem;
        }
      `}</style>
    </section>
  );
}

// CTA Section
function CTASection() {
  return (
    <section className="cta-section">
      <div className="cta-container">
        <div className="cta-content">
          <img src="/img/ailang-logo.svg" alt="AILANG" width="64" height="64" className="cta-logo" />
          <h2 className="cta-title">Ready to Build with AI?</h2>
          <p className="cta-subtitle">
            Start writing AILANG today. Full documentation, examples, and an interactive playground await.
          </p>
          <div className="cta-actions">
            <Link to="/docs/guides/getting-started" className="cta-btn cta-btn-primary">
              Get Started
              <ArrowRight size={18} />
            </Link>
            <Link to="/docs/playground" className="cta-btn cta-btn-secondary">
              Try Playground
            </Link>
          </div>
        </div>

        <div className="cta-links">
          <Link to="/docs/prompts" className="cta-link">
            <Cpu size={20} />
            <div>
              <strong>AI Teaching Prompt</strong>
              <span>Version {ACTIVE_PROMPT}</span>
            </div>
          </Link>
          <a href="https://www.sunholo.com/ailang-demos/" className="cta-link" target="_blank" rel="noopener noreferrer">
            <Globe size={20} />
            <div>
              <strong>Live Demos</strong>
              <span>9 interactive demos</span>
            </div>
          </a>
          <Link to="/docs/examples" className="cta-link">
            <Code2 size={20} />
            <div>
              <strong>Code Examples</strong>
              <span>66 examples</span>
            </div>
          </Link>
          <a href="https://github.com/sunholo-data/ailang" className="cta-link" target="_blank" rel="noopener noreferrer">
            <GitBranch size={20} />
            <div>
              <strong>GitHub</strong>
              <span>Source code</span>
            </div>
          </a>
        </div>
      </div>

      <style>{`
        .cta-section {
          padding: 6rem 2rem;
          background: linear-gradient(135deg, #0f1419 0%, #1a2332 100%);
          position: relative;
          overflow: hidden;
        }

        .cta-section::before {
          content: '';
          position: absolute;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: radial-gradient(circle at 50% 50%, rgba(231, 60, 23, 0.1) 0%, transparent 50%);
        }

        .cta-container {
          max-width: 1000px;
          margin: 0 auto;
          position: relative;
          z-index: 1;
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 4rem;
          align-items: center;
        }

        @media (max-width: 768px) {
          .cta-container {
            grid-template-columns: 1fr;
            gap: 3rem;
            text-align: center;
          }
        }

        .cta-logo {
          margin-bottom: 1rem;
          filter: drop-shadow(0 0 20px rgba(231, 60, 23, 0.3));
        }

        @media (max-width: 768px) {
          .cta-logo {
            margin: 0 auto 1rem;
            display: block;
          }
        }

        .cta-title {
          font-family: 'Montserrat', sans-serif;
          font-weight: 800;
          font-size: clamp(1.75rem, 4vw, 2.5rem);
          color: white;
          margin-bottom: 1rem;
        }

        .cta-subtitle {
          font-size: 1.1rem;
          color: rgba(255, 255, 255, 0.7);
          margin-bottom: 2rem;
          line-height: 1.7;
        }

        .cta-actions {
          display: flex;
          gap: 1rem;
          flex-wrap: wrap;
        }

        @media (max-width: 768px) {
          .cta-actions {
            justify-content: center;
          }
        }

        .cta-btn {
          display: inline-flex;
          align-items: center;
          gap: 0.5rem;
          padding: 0.875rem 1.5rem;
          font-family: 'Montserrat', sans-serif;
          font-weight: 600;
          font-size: 1rem;
          border-radius: 10px;
          transition: all 0.3s ease;
          text-decoration: none;
        }

        .cta-btn-primary {
          background: linear-gradient(135deg, #e73c17, #dd6b20);
          color: white;
        }

        .cta-btn-primary:hover {
          transform: translateY(-2px);
          box-shadow: 0 8px 25px rgba(231, 60, 23, 0.4);
          color: white;
        }

        .cta-btn-secondary {
          background: rgba(255, 255, 255, 0.1);
          color: white;
          border: 1px solid rgba(255, 255, 255, 0.2);
        }

        .cta-btn-secondary:hover {
          background: rgba(255, 255, 255, 0.15);
          color: white;
        }

        .cta-links {
          display: flex;
          flex-direction: column;
          gap: 1rem;
        }

        .cta-link {
          display: flex;
          align-items: center;
          gap: 1rem;
          padding: 1rem 1.25rem;
          background: rgba(255, 255, 255, 0.05);
          border: 1px solid rgba(255, 255, 255, 0.1);
          border-radius: 12px;
          color: white;
          text-decoration: none;
          transition: all 0.3s ease;
        }

        .cta-link:hover {
          background: rgba(255, 255, 255, 0.1);
          border-color: rgba(231, 60, 23, 0.5);
          transform: translateX(5px);
          color: white;
        }

        .cta-link svg {
          color: #ff5a3c;
          flex-shrink: 0;
        }

        .cta-link div {
          display: flex;
          flex-direction: column;
        }

        .cta-link strong {
          font-family: 'Montserrat', sans-serif;
          font-weight: 600;
          font-size: 1rem;
        }

        .cta-link span {
          font-size: 0.85rem;
          color: rgba(255, 255, 255, 0.5);
        }
      `}</style>
    </section>
  );
}

// Workflow Section - "What you actually do"
function WorkflowSection() {
  const steps = [
    {
      n: '1',
      icon: Bot,
      title: 'Agent writes .ail',
      desc: 'Plug AILANG into Claude Code, Codex, or any agent that follows the teaching prompt.',
    },
    {
      n: '2',
      icon: Shield,
      title: 'AILANG checks',
      desc: 'Types, effects, capabilities, and inline contracts are verified before anything runs.',
    },
    {
      n: '3',
      icon: Play,
      title: 'Run deterministically',
      desc: 'Same input → same output. Capabilities (`--caps IO,FS`) constrain what code is allowed to do.',
    },
    {
      n: '4',
      icon: Activity,
      title: 'Traces explain failures',
      desc: 'Structured per-effect traces feed back to the agent so the next iteration is targeted, not guessing.',
    },
  ];

  return (
    <section className="workflow-section">
      <div className="workflow-container">
        <div className="workflow-header">
          <h2 className="workflow-title">How AILANG works</h2>
          <p className="workflow-subtitle">
            Humans specify intent. Agents write <code>.ail</code>. AILANG constrains, runs, and explains.
          </p>
        </div>
        <div className="workflow-grid">
          {steps.map((s) => (
            <div key={s.n} className="workflow-step">
              <div className="workflow-step-head">
                <span className="workflow-step-n">{s.n}</span>
                <s.icon size={20} className="workflow-step-icon" />
              </div>
              <h3 className="workflow-step-title">{s.title}</h3>
              <p className="workflow-step-desc">{s.desc}</p>
            </div>
          ))}
        </div>
      </div>
      <style>{`
        .workflow-section {
          padding: 4rem 2rem;
          background: var(--ifm-background-color);
          border-bottom: 1px solid var(--ifm-color-emphasis-200);
        }
        .workflow-container {
          max-width: 1200px;
          margin: 0 auto;
        }
        .workflow-header {
          text-align: center;
          margin-bottom: 2.5rem;
        }
        .workflow-title {
          font-size: 2rem;
          font-weight: 700;
          margin: 0 0 0.5rem 0;
          color: var(--ifm-color-content);
        }
        .workflow-subtitle {
          font-size: 1.05rem;
          color: var(--ifm-color-content-secondary);
          margin: 0;
        }
        .workflow-subtitle code {
          background: var(--ifm-code-background);
          padding: 0.1rem 0.35rem;
          border-radius: 4px;
          font-size: 0.95em;
        }
        .workflow-grid {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
          gap: 1.25rem;
        }
        @media (max-width: 900px) {
          .workflow-grid { grid-template-columns: repeat(2, 1fr); }
        }
        @media (max-width: 540px) {
          .workflow-grid { grid-template-columns: 1fr; }
        }
        .workflow-step {
          padding: 1.5rem;
          border: 1px solid var(--ifm-color-emphasis-200);
          border-radius: 12px;
          background: var(--ifm-card-background-color, var(--ifm-background-surface-color));
          transition: transform 0.15s ease, border-color 0.15s ease;
        }
        .workflow-step:hover {
          transform: translateY(-2px);
          border-color: var(--ifm-color-primary);
        }
        .workflow-step-head {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 0.75rem;
        }
        .workflow-step-n {
          font-family: 'JetBrains Mono', monospace;
          font-size: 0.85rem;
          font-weight: 700;
          color: var(--ifm-color-primary);
          background: color-mix(in srgb, var(--ifm-color-primary) 12%, transparent);
          width: 1.75rem;
          height: 1.75rem;
          border-radius: 50%;
          display: inline-flex;
          align-items: center;
          justify-content: center;
        }
        .workflow-step-icon {
          color: var(--ifm-color-content-secondary);
        }
        .workflow-step-title {
          font-size: 1.05rem;
          font-weight: 600;
          margin: 0 0 0.4rem 0;
          color: var(--ifm-color-content);
        }
        .workflow-step-desc {
          font-size: 0.9rem;
          line-height: 1.5;
          color: var(--ifm-color-content-secondary);
          margin: 0;
        }
        .workflow-step-desc code {
          background: var(--ifm-code-background);
          padding: 0.05rem 0.3rem;
          border-radius: 3px;
          font-size: 0.95em;
        }
      `}</style>
    </section>
  );
}

// Main Page Component
export default function Home() {
  const {siteConfig} = useDocusaurusContext();

  return (
    <Layout
      title="AI-First Programming Language"
      description="AILANG is a deterministic, effect-typed programming language designed as a safe target for AI-generated programs.">
      <main>
        <HeroSection />
        <WorkflowSection />
        <QuickStartSection />
        <FeaturesSection />
        <BenchmarkMini />
        <AddToAgent />
        <DemosShowcase />
        <CTASection />
      </main>
    </Layout>
  );
}
