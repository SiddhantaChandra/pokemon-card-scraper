import { SWRConfig } from 'swr';
import { DarkModeProvider } from '../lib/darkModeContext';
import "@/styles/globals.css";

// SWR configuration
const swrConfig = {
  // Global error handler
  onError: (error, key) => {
    console.error('SWR Error:', error, 'Key:', key);
  },
  
  // Global options
  revalidateOnFocus: false,
  revalidateOnReconnect: true,
  refreshInterval: 0,
  dedupingInterval: 2000,
  
  // Error retry configuration
  errorRetryCount: 3,
  errorRetryInterval: 1000,
  
  // Loading timeout
  loadingTimeout: 3000,
  
  // Keep previous data when key changes
  keepPreviousData: true,
};

export default function App({ Component, pageProps }) {
  return (
    <DarkModeProvider>
      <SWRConfig value={swrConfig}>
        <Component {...pageProps} />
      </SWRConfig>
    </DarkModeProvider>
  );
}
