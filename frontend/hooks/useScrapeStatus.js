import useSWR from 'swr';
import { useState, useCallback } from 'react';
import { scrapeAPI, statsAPI } from '../lib/api';

// Custom hook for monitoring scrape status
export function useScrapeStatus() {
  const [isStarting, setIsStarting] = useState(false);
  const [isStopping, setIsStopping] = useState(false);
  const [isPausing, setIsPausing] = useState(false);
  const [isResuming, setIsResuming] = useState(false);

  // Fetch scrape status with polling when running
  const {
    data: scrapeData,
    error: scrapeError,
    isLoading: scrapeLoading,
    mutate: mutateScrape
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

  // Fetch database stats
  const {
    data: statsData,
    error: statsError,
    isLoading: statsLoading,
    mutate: mutateStats
  } = useSWR(
    'database-stats',
    () => statsAPI.getStats().then(res => res.data),
    {
      refreshInterval: 30000, // Refresh every 30 seconds
      revalidateOnFocus: false,
      errorRetryCount: 3,
    }
  );

  // Start scraping
  const startScraping = useCallback(async (options = {}) => {
    setIsStarting(true);
    try {
      await scrapeAPI.start(options);
      // Refresh both scrape status and stats
      mutateScrape();
      mutateStats();
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
  }, [mutateScrape, mutateStats]);

  // Stop scraping
  const stopScraping = useCallback(async () => {
    setIsStopping(true);
    try {
      await scrapeAPI.stop();
      // Refresh both scrape status and stats
      mutateScrape();
      mutateStats();
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
  }, [mutateScrape, mutateStats]);

  // Restart scraping
  const restartScraping = useCallback(async (options = {}) => {
    setIsStarting(true);
    try {
      await scrapeAPI.restart(options);
      // Refresh both scrape status and stats
      mutateScrape();
      mutateStats();
      return { success: true };
    } catch (error) {
      console.error('Failed to restart scraping:', error);
      return { 
        success: false, 
        error: error.message || 'Failed to restart scraping' 
      };
    } finally {
      setIsStarting(false);
    }
  }, [mutateScrape, mutateStats]);

  // Pause scraping
  const pauseScraping = useCallback(async () => {
    setIsPausing(true);
    try {
      await scrapeAPI.pause();
      // Refresh scrape status
      mutateScrape();
      return { success: true };
    } catch (error) {
      console.error('Failed to pause scraping:', error);
      return { 
        success: false, 
        error: error.message || 'Failed to pause scraping' 
      };
    } finally {
      setIsPausing(false);
    }
  }, [mutateScrape]);

  // Resume scraping
  const resumeScraping = useCallback(async () => {
    setIsResuming(true);
    try {
      await scrapeAPI.resume();
      // Refresh scrape status
      mutateScrape();
      return { success: true };
    } catch (error) {
      console.error('Failed to resume scraping:', error);
      return { 
        success: false, 
        error: error.message || 'Failed to resume scraping' 
      };
    } finally {
      setIsResuming(false);
    }
  }, [mutateScrape]);

  // Calculate progress percentage
  const getProgressPercentage = useCallback(() => {
    if (!scrapeData || !scrapeData.running || !scrapeData.total_pages) return 0;
    return Math.round((scrapeData.current_page / scrapeData.total_pages) * 100);
  }, [scrapeData]);

  // Get estimated time remaining
  const getEstimatedTimeRemaining = useCallback(() => {
    if (!scrapeData || !scrapeData.running || !scrapeData.current_page || !scrapeData.total_pages) {
      return null;
    }

    const pagesRemaining = scrapeData.total_pages - scrapeData.current_page;
    const startTime = new Date(scrapeData.start_time);
    const now = new Date();
    const elapsedMs = now - startTime;
    const pagesCompleted = scrapeData.current_page;
    
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
  }, [scrapeData]);

  // Format last updated time
  const getLastUpdatedText = useCallback(() => {
    if (!scrapeData?.last_updated) return 'Never';
    
    const lastUpdated = new Date(scrapeData.last_updated);
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
  }, [scrapeData]);

  // Refresh both data sources
  const refresh = useCallback(() => {
    mutateScrape();
    mutateStats();
  }, [mutateScrape, mutateStats]);

  return {
    // Scraper status data
    isRunning: scrapeData?.running || false,
    isPaused: scrapeData?.paused || false,
    currentPage: scrapeData?.current_page || 0,
    totalPages: scrapeData?.total_pages || 0,
    itemsScraped: scrapeData?.items_scraped || 0,
    cardsPerMinute: scrapeData?.cards_per_minute || 0,
    estimatedTimeRemainingBackend: scrapeData?.estimated_time_remaining,
    startTime: scrapeData?.start_time,
    lastUpdated: scrapeData?.last_updated,
    pausedAt: scrapeData?.paused_at,
    
    // Database stats
    totalCardsInDatabase: statsData?.database?.total_cards || 0,
    inStockCardsInDatabase: statsData?.database?.in_stock_cards || 0,
    priceRange: statsData?.database?.price_range || { min: 0, max: 0 },
    
    // Calculated values
    progressPercentage: scrapeData?.progress_percent || getProgressPercentage(),
    estimatedTimeRemaining: getEstimatedTimeRemaining(),
    lastUpdatedText: getLastUpdatedText(),
    
    // Loading states
    isLoading: scrapeLoading || statsLoading,
    isStarting,
    isStopping,
    isPausing,
    isResuming,
    error: scrapeError || statsError,
    
    // Actions
    startScraping,
    stopScraping,
    pauseScraping,
    resumeScraping,
    restartScraping,
    refresh,
    
    // Helper methods
    canStart: !scrapeData?.running && !isStarting,
    canStop: scrapeData?.running && !isStopping && !scrapeData?.paused,
    canPause: scrapeData?.running && !scrapeData?.paused && !isPausing,
    canResume: scrapeData?.running && scrapeData?.paused && !isResuming,
    canRestart: !isStarting,
  };
} 