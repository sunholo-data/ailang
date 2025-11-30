import React from 'react';
import './Breadcrumb.css';

interface BreadcrumbItem {
  label: string;
  onClick?: () => void;
}

interface BreadcrumbProps {
  items: BreadcrumbItem[];
}

export const Breadcrumb: React.FC<BreadcrumbProps> = ({ items }) => {
  return (
    <nav className="breadcrumb">
      {items.map((item, index) => (
        <React.Fragment key={index}>
          {index > 0 && <span className="breadcrumb-separator">/</span>}
          {item.onClick ? (
            <button className="breadcrumb-link" onClick={item.onClick}>
              {item.label}
            </button>
          ) : (
            <span className="breadcrumb-current">{item.label}</span>
          )}
        </React.Fragment>
      ))}
    </nav>
  );
};

export default Breadcrumb;
