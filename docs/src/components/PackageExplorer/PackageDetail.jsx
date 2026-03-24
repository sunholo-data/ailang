import React from 'react';

/**
 * PackageDetail — renders on static package pages.
 * Shows static data immediately, will hydrate with live data in M3.
 */
export default function PackageDetail({ packageName, staticData }) {
  if (!staticData) {
    return <p>Loading package details for {packageName}...</p>;
  }

  const effects = staticData.effects && staticData.effects.length > 0
    ? staticData.effects.join(', ')
    : 'Pure';

  return (
    <div>
      <h3>Exports</h3>
      {staticData.exports && staticData.exports.length > 0 ? (
        <ul>
          {staticData.exports.map((exp) => (
            <li key={exp}><code>{exp}</code></li>
          ))}
        </ul>
      ) : (
        <p>No exports listed.</p>
      )}

      {staticData.dependencies && staticData.dependencies.length > 0 && (
        <>
          <h3>Dependencies</h3>
          <ul>
            {staticData.dependencies.map((dep) => (
              <li key={dep}><code>{dep}</code></li>
            ))}
          </ul>
        </>
      )}

      <p><em>Version timeline and provenance details will load from the registry API.</em></p>
    </div>
  );
}
