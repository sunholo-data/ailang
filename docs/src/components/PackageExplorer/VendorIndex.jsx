import React from 'react';
import styles from './styles.module.css';

/**
 * VendorIndex — card grid listing all packages for a vendor.
 * Rendered on generated vendor index pages (e.g., /docs/packages/sunholo/).
 */
export default function VendorIndex({ vendor, packages = [] }) {
  if (!packages || packages.length === 0) {
    return <p>No packages found for {vendor}.</p>;
  }

  return (
    <div className={styles.vendorGrid}>
      {packages.map((pkg) => {
        const shortName = pkg.name.split('/')[1];
        const effects = pkg.effects && pkg.effects.length > 0
          ? pkg.effects.join(', ')
          : 'Pure';

        return (
          <a
            key={pkg.name}
            href={`./${shortName}`}
            className={styles.packageCard}
          >
            <div className={styles.cardHeader}>
              <span className={styles.packageName}>{shortName}</span>
              <span className={styles.version}>v{pkg.latest}</span>
            </div>
            <p className={styles.summary}>{pkg.ai_summary || 'No description'}</p>
            <div className={styles.badges}>
              <span className={`${styles.badge} ${styles.stabilityBadge}`}>
                {pkg.stability || 'experimental'}
              </span>
              <span className={`${styles.badge} ${styles.effectBadge}`}>
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
