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
  Play
} from 'lucide-react';
import { STABLE_RELEASE, ACTIVE_PROMPT } from '../constants/version';

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
          AILANG is a pure functional language designed from the ground up for
          AI-assisted development. Explicit effects, deterministic execution,
          and structured traces for model training.
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
        </div>

        <div className="hero-code">
          <div className="hero-code-header">
            <span className="hero-code-dot red" />
            <span className="hero-code-dot yellow" />
            <span className="hero-code-dot green" />
            <span className="hero-code-filename">hello.ail</span>
          </div>
          <pre className="hero-code-content">
{`module examples/hello

effect IO

let main: () -> ! IO Unit =
  let _ = print("Hello, AILANG!")
  ()`}
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

// Features Section
const features = [
  {
    icon: Zap,
    title: 'Pure Functional',
    description: 'Immutable data structures and first-class functions. Everything is an expression that returns a value.',
    color: '#e73c17'
  },
  {
    icon: Shield,
    title: 'Algebraic Effects',
    description: 'Track all side effects in the type system. IO, FS, Clock, Net with capability-based security.',
    color: '#2c7a7b'
  },
  {
    icon: Layers,
    title: 'Type Classes',
    description: 'Num, Eq, Ord, Show with dictionary-passing semantics. Auto-imported prelude for zero boilerplate.',
    color: '#6b46c1'
  },
  {
    icon: Code2,
    title: 'Pattern Matching',
    description: 'Exhaustive pattern matching on ADTs, tuples, lists, and records with guards and wildcards.',
    color: '#2b6cb0'
  },
  {
    icon: Bot,
    title: 'AI-Optimized',
    description: 'Generate structured execution traces for model training. Deterministic semantics for AI reasoning.',
    color: '#dd6b20'
  },
  {
    icon: Terminal,
    title: 'Go Codegen',
    description: 'Compile AILANG to typed Go code. Multi-file support, effect handlers, and full type safety.',
    color: '#38a169'
  }
];

function FeaturesSection() {
  return (
    <section className="features-section">
      <div className="features-container">
        <div className="features-header">
          <h2 className="features-title">Built for the AI Era</h2>
          <p className="features-subtitle">
            Every language feature considers AI tooling needs. Deterministic, traceable, and safe by design.
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
          max-width: 600px;
          margin: 0 auto;
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

// CTA Section
function CTASection() {
  return (
    <section className="cta-section">
      <div className="cta-container">
        <div className="cta-content">
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

// Main Page Component
export default function Home() {
  const {siteConfig} = useDocusaurusContext();

  return (
    <Layout
      title="AI-First Programming Language"
      description="AILANG is a pure functional programming language designed for AI-assisted software development">
      <main>
        <HeroSection />
        <FeaturesSection />
        <CTASection />
      </main>
    </Layout>
  );
}
