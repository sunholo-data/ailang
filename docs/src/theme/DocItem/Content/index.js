import React from 'react';
import Content from '@theme-original/DocItem/Content';
import { useDoc } from '@docusaurus/plugin-content-docs/client';
import Link from '@docusaurus/Link';

const TIER_CONFIG = {
  experimental: {
    className: 'admonition admonition-caution alert alert--warning',
    badge: 'Experimental',
    message:
      'This surface is experimental and may change between minor releases. See ',
    linkLabel: 'Stability Promise (1.x)',
    linkTo: '/docs/reference/stability',
    suffix: ' for tier guarantees.',
  },
  internal: {
    className: 'admonition admonition-info alert alert--info',
    badge: 'Internal / Operator Docs',
    message:
      'This page documents internal AILANG contributor or fleet tooling rather than the stable 1.x language API. Start at ',
    linkLabel: 'Getting Started',
    linkTo: '/docs/guides/getting-started',
    suffix: ' for user-facing docs.',
  },
};

function TierBanner({ tier }) {
  const cfg = TIER_CONFIG[tier];
  if (!cfg) {
    return null;
  }
  return (
    <div
      className={cfg.className}
      role="note"
      style={{
        marginBottom: '1.25rem',
        padding: '0.75rem 1rem',
        borderRadius: '8px',
        fontSize: '0.92rem',
      }}
    >
      <strong>{cfg.badge}:</strong> {cfg.message}
      <Link to={cfg.linkTo}>{cfg.linkLabel}</Link>
      {cfg.suffix}
    </div>
  );
}

export default function ContentWrapper(props) {
  const { frontMatter } = useDoc();
  const tier = typeof frontMatter?.tier === 'string' ? frontMatter.tier.toLowerCase() : null;

  return (
    <>
      <TierBanner tier={tier} />
      <Content {...props} />
    </>
  );
}
