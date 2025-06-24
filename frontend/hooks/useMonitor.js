import { useState, useCallback } from 'react';
import useSWR from 'swr';
import { monitorAPI } from '../lib/api';

// Custom hook for managing monitor state
export const useMonitor = () => {
  const [isSubmitting, setIsSubmitting] = useState(false);

  // SWR for monitor status
  const {
    data: status,
    error: statusError,
    mutate: mutateStatus,
    isLoading: isLoadingStatus,
  } = useSWR('monitor-status', async () => {
    const response = await monitorAPI.getStatus();
    return response.data;
  }, {
    refreshInterval: 5000, // Refresh every 5 seconds
    revalidateOnFocus: true,
    revalidateOnReconnect: true,
  });

  // SWR for monitor stats
  const {
    data: stats,
    error: statsError,
    mutate: mutateStats,
    isLoading: isLoadingStats,
  } = useSWR('monitor-stats', async () => {
    const response = await monitorAPI.getStats();
    return response.data;
  }, {
    refreshInterval: 10000, // Refresh every 10 seconds
    revalidateOnFocus: true,
    revalidateOnReconnect: true,
  });

  // Start monitoring function
  const startMonitoring = useCallback(async () => {
    try {
      setIsSubmitting(true);
      const response = await monitorAPI.start();
      
      // Update cache immediately
      await Promise.all([mutateStatus(), mutateStats()]);
      
      return response.data;
    } catch (error) {
      console.error('Failed to start monitoring:', error);
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  }, [mutateStatus, mutateStats]);

  // Stop monitoring function
  const stopMonitoring = useCallback(async () => {
    try {
      setIsSubmitting(true);
      const response = await monitorAPI.stop();
      
      // Update cache immediately
      await Promise.all([mutateStatus(), mutateStats()]);
      
      return response.data;
    } catch (error) {
      console.error('Failed to stop monitoring:', error);
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  }, [mutateStatus, mutateStats]);

  // Refresh both status and stats
  const refresh = useCallback(() => {
    return Promise.all([mutateStatus(), mutateStats()]);
  }, [mutateStatus, mutateStats]);

  // Helper to determine if monitoring is running
  const isRunning = status === 'running';
  const isStopped = status === 'stopped';
  const isStarting = status === 'starting';
  const isStopping = status === 'stopping';
  const hasError = status === 'error';

  // Helper to get status display info
  const getStatusInfo = () => {
    switch (status) {
      case 'running':
        return {
          text: 'Running',
          color: 'text-green-600 dark:text-green-400',
          bgColor: 'bg-green-100 dark:bg-green-900',
          icon: '🟢',
        };
      case 'stopped':
        return {
          text: 'Stopped',
          color: 'text-gray-600 dark:text-gray-400',
          bgColor: 'bg-gray-100 dark:bg-gray-800',
          icon: '⭕',
        };
      case 'starting':
        return {
          text: 'Starting...',
          color: 'text-blue-600 dark:text-blue-400',
          bgColor: 'bg-blue-100 dark:bg-blue-900',
          icon: '🔄',
        };
      case 'stopping':
        return {
          text: 'Stopping...',
          color: 'text-orange-600 dark:text-orange-400',
          bgColor: 'bg-orange-100 dark:bg-orange-900',
          icon: '⏸️',
        };
      case 'error':
        return {
          text: 'Error',
          color: 'text-red-600 dark:text-red-400',
          bgColor: 'bg-red-100 dark:bg-red-900',
          icon: '❌',
        };
      default:
        return {
          text: 'Unknown',
          color: 'text-gray-600 dark:text-gray-400',
          bgColor: 'bg-gray-100 dark:bg-gray-800',
          icon: '❓',
        };
    }
  };

  return {
    // Data
    status,
    stats: stats || {
      total_trackers: 0,
      active_trackers: 0,
      checks_completed: 0,
      checks_failed: 0,
      last_check_time: null,
      next_check_time: null,
      notifications_sent: 0,
      errors_today: 0,
      average_check_time: 0,
      uptime_percentage: 0,
    },
    
    // Loading states
    isLoading: isLoadingStatus || isLoadingStats,
    isSubmitting,
    error: statusError || statsError,
    
    // Status helpers
    isRunning,
    isStopped,
    isStarting,
    isStopping,
    hasError,
    getStatusInfo,
    
    // Actions
    startMonitoring,
    stopMonitoring,
    refresh,
  };
};

export default useMonitor; 