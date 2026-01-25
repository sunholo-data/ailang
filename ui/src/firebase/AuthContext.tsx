// Authentication context for Firebase Auth
import React, { createContext, useContext, useEffect, useState } from 'react';
import { User, signInWithPopup, signOut, onAuthStateChanged } from 'firebase/auth';
import { auth, googleProvider, isFirebaseConfigured } from './config';

// User role from backend
interface UserRole {
  role: string; // "Approver" | "Viewer" | "Guest"
  workspaceId: string;
}

interface AuthContextType {
  user: User | null;
  userRole: UserRole | null;
  loading: boolean;
  isConfigured: boolean;
  signInWithGoogle: () => Promise<void>;
  logout: () => Promise<void>;
  getIdToken: () => Promise<string | null>;
  isApprover: () => boolean;
  isViewer: () => boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [userRole, setUserRole] = useState<UserRole | null>(null);
  const [loading, setLoading] = useState(true);
  const isConfigured = isFirebaseConfigured();

  useEffect(() => {
    if (!isConfigured || !auth) {
      setLoading(false);
      return;
    }

    const unsubscribe = onAuthStateChanged(auth, async (firebaseUser) => {
      setUser(firebaseUser);

      if (firebaseUser) {
        // Fetch user role from backend
        try {
          const token = await firebaseUser.getIdToken();
          const response = await fetch('/api/auth/whoami', {
            headers: {
              'Authorization': `Bearer ${token}`,
            },
          });

          if (response.ok) {
            const data = await response.json();
            setUserRole({
              role: data.role || 'Guest',
              workspaceId: data.workspace_id || 'default',
            });
          } else {
            setUserRole({ role: 'Guest', workspaceId: 'default' });
          }
        } catch (error) {
          console.error('Failed to fetch user role:', error);
          setUserRole({ role: 'Guest', workspaceId: 'default' });
        }
      } else {
        setUserRole(null);
      }

      setLoading(false);
    });

    return () => unsubscribe();
  }, [isConfigured]);

  const signInWithGoogle = async () => {
    if (!auth) {
      console.error('Firebase not configured');
      return;
    }
    try {
      await signInWithPopup(auth, googleProvider);
    } catch (error) {
      console.error('Sign in error:', error);
      throw error;
    }
  };

  const logout = async () => {
    if (!auth) return;
    try {
      await signOut(auth);
      setUserRole(null);
    } catch (error) {
      console.error('Sign out error:', error);
      throw error;
    }
  };

  const getIdToken = async (): Promise<string | null> => {
    if (!user) return null;
    try {
      return await user.getIdToken();
    } catch (error) {
      console.error('Failed to get ID token:', error);
      return null;
    }
  };

  const isApprover = () => userRole?.role === 'Approver';
  const isViewer = () => userRole?.role === 'Viewer' || userRole?.role === 'Approver';

  const value = {
    user,
    userRole,
    loading,
    isConfigured,
    signInWithGoogle,
    logout,
    getIdToken,
    isApprover,
    isViewer,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
