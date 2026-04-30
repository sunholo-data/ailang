import React from 'react';
import {
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Info,
  Lightbulb,
  Code2,
  Zap,
  Bot,
  User,
  Wrench,
  Rocket,
  Target,
  BookOpen,
  Scale,
  Brain,
  Play,
  Box,
  BarChart,
  Shield,
  Layers,
  Gamepad2,
  Server,
  Terminal,
  Database,
  Globe,
  X
} from 'lucide-react';

// GithubMark is an inline SVG of the GitHub Octocat mark. lucide-react v0.5+
// removed brand icons (Github, Twitter, etc.); we ship our own to keep the
// API stable. Path data is the canonical Octocat from
// https://github.com/logos (CC-BY).
function GithubMark({ size = 16, className = '', style }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="currentColor"
      aria-hidden="true"
      className={className}
      style={style}
    >
      <path d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.55 0-.27-.01-.99-.01-1.94-3.2.7-3.88-1.54-3.88-1.54-.52-1.34-1.27-1.7-1.27-1.7-1.04-.71.08-.7.08-.7 1.15.08 1.76 1.18 1.76 1.18 1.02 1.75 2.68 1.24 3.34.95.1-.74.4-1.24.73-1.53-2.55-.29-5.24-1.28-5.24-5.69 0-1.26.45-2.29 1.18-3.1-.12-.29-.51-1.46.11-3.04 0 0 .96-.31 3.15 1.18.91-.25 1.89-.38 2.87-.39.97.01 1.96.14 2.87.39 2.19-1.49 3.15-1.18 3.15-1.18.62 1.58.23 2.75.11 3.04.74.81 1.18 1.84 1.18 3.1 0 4.42-2.69 5.4-5.25 5.68.41.36.78 1.06.78 2.13 0 1.54-.01 2.78-.01 3.16 0 .31.21.67.8.55C20.71 21.39 24 17.08 24 12 24 5.65 18.85.5 12.5.5H12z" />
    </svg>
  );
}

const iconMap = {
  check: CheckCircle2,
  cross: XCircle,
  warning: AlertTriangle,
  info: Info,
  idea: Lightbulb,
  code: Code2,
  zap: Zap,
  bot: Bot,
  user: User,
  wrench: Wrench,
  rocket: Rocket,
  target: Target,
  book: BookOpen,
  scale: Scale,
  brain: Brain,
  play: Play,
  box: Box,
  'bar-chart': BarChart,
  github: GithubMark,
  shield: Shield,
  layers: Layers,
  gamepad: Gamepad2,
  server: Server,
  terminal: Terminal,
  database: Database,
  globe: Globe,
  x: X,
};

export default function Icon({ name, size = 16, className = '', inline = false, color }) {
  const IconComponent = iconMap[name];

  if (!IconComponent) {
    console.warn(`Icon "${name}" not found`);
    return null;
  }

  const style = {
    display: inline ? 'inline' : 'inline-block',
    verticalAlign: 'middle',
    marginRight: inline ? '0.25em' : undefined,
  };

  if (color) {
    style.color = color;
  }

  return (
    <IconComponent
      size={size}
      className={className}
      style={style}
    />
  );
}

// Convenience components for common cases
export function CheckIcon(props) {
  return <Icon name="check" color="var(--ifm-color-success)" {...props} />;
}

export function CrossIcon(props) {
  return <Icon name="cross" color="var(--ifm-color-danger)" {...props} />;
}

export function InfoIcon(props) {
  return <Icon name="info" color="var(--ifm-color-info)" {...props} />;
}

export function WarningIcon(props) {
  return <Icon name="warning" color="var(--ifm-color-warning)" {...props} />;
}
