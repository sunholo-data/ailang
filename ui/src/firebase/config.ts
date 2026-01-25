// Firebase configuration for AILANG Dashboard
// Get these values from Firebase Console > Project Settings > Your Apps > Web App
import { initializeApp } from 'firebase/app';
import { getAuth, GoogleAuthProvider } from 'firebase/auth';

// Firebase configuration from environment or defaults
// In production, set these via environment variables
const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY || '',
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN || 'ailang-dev.firebaseapp.com',
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID || 'ailang-dev',
  storageBucket: import.meta.env.VITE_FIREBASE_STORAGE_BUCKET || 'ailang-dev.firebasestorage.app',
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID || '',
  appId: import.meta.env.VITE_FIREBASE_APP_ID || '',
};

// Initialize Firebase (only if API key is configured)
let app: ReturnType<typeof initializeApp> | null = null;
let auth: ReturnType<typeof getAuth> | null = null;

// Check if Firebase is configured
export const isFirebaseConfigured = () => {
  return !!firebaseConfig.apiKey;
};

if (firebaseConfig.apiKey) {
  app = initializeApp(firebaseConfig);
  auth = getAuth(app);
}

// Google Auth Provider
export const googleProvider = new GoogleAuthProvider();

export { app, auth };
export default firebaseConfig;
