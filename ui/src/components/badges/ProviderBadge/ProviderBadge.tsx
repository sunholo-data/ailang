import styles from './ProviderBadge.module.css';

interface ProviderBadgeProps {
  provider: string;
  size?: 'small' | 'medium';
}

// Map provider IDs to display names and colors
const providerConfig: Record<string, { name: string; colorClass: string }> = {
  'claude': { name: 'Claude', colorClass: 'claude' },
  'claude-code': { name: 'Claude', colorClass: 'claude' },
  'anthropic': { name: 'Claude', colorClass: 'claude' },
  'gemini': { name: 'Gemini', colorClass: 'gemini' },
  'gemini-cli': { name: 'Gemini', colorClass: 'gemini' },
  'google': { name: 'Gemini', colorClass: 'gemini' },
  'openai': { name: 'GPT', colorClass: 'openai' },
  'gpt': { name: 'GPT', colorClass: 'openai' },
  'gpt-5': { name: 'GPT-5', colorClass: 'openai' },
  'ollama': { name: 'Ollama', colorClass: 'ollama' },
};

export function ProviderBadge({ provider, size = 'small' }: ProviderBadgeProps) {
  if (!provider) {
    return null;
  }

  const normalizedProvider = provider.toLowerCase();
  const config = providerConfig[normalizedProvider] || { name: provider, colorClass: 'default' };

  return (
    <span
      className={`${styles.badge} ${styles[config.colorClass]} ${styles[size]}`}
      title={`Provider: ${provider}`}
    >
      {config.name}
    </span>
  );
}
