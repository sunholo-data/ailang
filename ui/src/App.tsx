import React from 'react';
import { AuthProvider } from './firebase';
import { ControlPlane } from './features/controlplane/ControlPlane';
import './App.css';

export const App: React.FC = () => {
  return (
    <AuthProvider>
      <ControlPlane />
    </AuthProvider>
  );
};
