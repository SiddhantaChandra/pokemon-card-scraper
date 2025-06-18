import useSWR from 'swr';
import { useState, useCallback } from 'react';
import { scrapeAPI } from '../lib/api';

// Custom hook for monitoring scrape status
export function useScrapeStatus() {
  const [isStarting, setIsStarting] = useState(false);
  const [isStopping, setIsStopping] = useState(false);

  // Fetch scrape status with polling when running
  const {
    data,
    error,
    isLoading,
    mutate
  } = useSWR(
    'scrape-status',
    () => scrapeAPI.getStatus().then(res => res.data),
    {
      // Poll every 2 seconds when scraping is running
      refreshInterval: (data) => data?.running ? 2000 : 0,
      revalidateOnFocus: false,
      errorRetryCount: 3,
    }
  );

  // Start scraping
  const startScraping = useCallback(async (options = {}) => {
    setIsStarting(true);
    try {
      await scrapeAPI.start(options);
      // Refresh status immediately
      mutate();
      return { success: true };
    } catch (error) {
      console.error('Failed to start scraping:', error);
      return { 
        success: false, 
        error: error.message || 'Failed to start scraping' 
      };
    } finally {
      setIsStarting(false);
    }
  }, [mutate]);

  // Stop scraping
  const stopScraping = useCallback(async () => {
    setIsStopping(true);
    try {
      await scrapeAPI.stop();
      // Refresh status immediately
      mutate();
      return { success: true };
    } catch (error) {
      console.error('Failed to stop scraping:', error);
      return { 
        success: false, 
        error: error.message || 'Failed to stop scraping' 
      };
    } finally {
      setIsStopping(false);
    }
  }, [mutate]);

  // Calculate progress percentage
  const getProgressPercentage = useCallback(() => {
    if (!data || !data.running || !data.total_pages) return 0;
    return Math.round((data.current_page / data.total_pages) * 100);
  }, [data]);

  // Get estimated time remaining
  const getEstimatedTimeRemaining = useCallback(() => {
    if (!data || !data.running || !data.current_page || !data.total_pages) {
      return null;
    }

    const pagesRemaining = data.total_pages - data.current_page;
    const startTime = new Date(data.start_time);
    const now = new Date();
    const elapsedMs = now - startTime;
    const pagesCompleted = data.current_page;
    
    if (pagesCompleted === 0) return null;

    const avgTimePerPage = elapsedMs / pagesCompleted;
    const estimatedRemainingMs = pagesRemaining * avgTimePerPage;

    // Convert to human readable format
    const remainingMinutes = Math.ceil(estimatedRemainingMs / (1000 * 60));
    
    if (remainingMinutes < 1) {
      return 'Less than 1 minute';
    } else if (remainingMinutes === 1) {
      return '1 minute';
    } else {
      return `${remainingMinutes} minutes`;
    }
  }, [data]);

  // Format last updated time
  const getLastUpdatedText = useCallback(() => {
    if (!data?.last_updated) return 'Never';
    
    const lastUpdated = new Date(data.last_updated);
    const now = new Date();
    const diffMs = now - lastUpdated;
    const diffSeconds = Math.floor(diffMs / 1000);
    
    if (diffSeconds < 60) {
      return `${diffSeconds} seconds ago`;
    } else if (diffSeconds < 3600) {
      const minutes = Math.floor(diffSeconds / 60);
      return `${minutes} minute${minutes === 1 ? '' : 's'} ago`;
    } else {
      return lastUpdated.toLocaleTimeString();
    }
  }, [data]);

  return {
    // Status data
    isRunning: data?.running || false,
    currentPage: data?.current_page || 0,
    totalPages: data?.total_pages || 0,
    itemsScraped: data?.items_scraped || 0,
    startTime: data?.start_time,
    lastUpdated: data?.last_updated,
    
    // Calculated values
    progressPercentage: getProgressPercentage(),
    estimatedTimeRemaining: getEstimatedTimeRemaining(),
    lastUpdatedText: getLastUpdatedText(),
    
    // Loading states
    isLoading,
    isStarting,
    isStopping,
    error,
    
    // Actions
    startScraping,
    stopScraping,
    refresh: mutate,
    
    // Helper methods
    canStart: !data?.running && !isStarting,
    canStop: data?.running && !isStopping,
  };
} 