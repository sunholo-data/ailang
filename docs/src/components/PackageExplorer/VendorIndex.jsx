import React from 'react';
import styles from './styles.module.css';

/**
 * VendorIndex — card grid listing all packages for a vendor.
 * Rendered on generated vendor index pages (e.g., /docs/packages/sunholo/).
 */
export default function VendorIndex({ vendor, packages = [] }) {
  if (!packages || packages.length === 0) {
    return (
      <div className={styles.emptyState}>
        <p>No packages found for <strong>{vendor}</strong>.</p>
      </div>
    );
  }

  return (
    <div className={styles.vendorGrid}>
      {packages.map((pkg) => {
        const shortName = pkg.name.split('/')[1];
        const isPure = !pkg.effects || pkg.effects.length === 0;
        const effects = isPure ? 'Pure' : pkg.effects.join(', ');

        const stabilityClass = pkg.stability === 'stable'
          ? styles.stabilityStable
          : pkg.stability === 'frozen'
            ? styles.stabilityFrozen
            : styles.stabilityBadge;

        return (
          <a
            key={pkg.name}
            href={`/docs/packages/${vendor}/${shortName}`}
            className={styles.packageCard}
          >
            <div className={styles.cardHeader}>
              <span className={styles.packageName}>{shortName}</span>
              <span className={styles.version}>v{pkg.latest}</span>
            </div>
            <p className={styles.summary}>{pkg.ai_summary || 'No description'}</p>
            <div className={styles.badges}>
              <span className={`${styles.badge} ${stabilityClass}`}>
                {pkg.stability || 'experimental'}
              </span>
              <span className={`${styles.badge} ${isPure ? styles.effectPure : styles.effectBadge}`}>
                {effects}
              </span>
              {(pkg.tags || []).slice(0, 3).map((tag) => (
                <span key={tag} className={`${styles.badge} ${styles.tagBadge}`}>
                  {tag}
                </span>
              ))}
            </div>
          </a>
        );
      })}
    </div>
  );
}
