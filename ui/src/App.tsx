import React from 'react';
import { AuthProvider } from './firebase';
import { ErrorBoundary } from './components/ErrorBoundary';
import { ControlPlane } from './features/controlplane/ControlPlane';
import './App.css';

export const App: React.FC = () => {
  return (
    <AuthProvider>
      <ErrorBoundary>
        <ControlPlane />
      </ErrorBoundary>
    </AuthProvider>
  );
};
