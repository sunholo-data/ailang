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
  Github
} from 'lucide-react';

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
  github: Github,
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
