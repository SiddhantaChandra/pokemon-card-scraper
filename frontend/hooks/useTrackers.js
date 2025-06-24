import { useState, useCallback } from 'react';
import useSWR from 'swr';
import { trackerAPI } from '../lib/api';

// Custom hook for managing tracker state
export const useTrackers = (initialFilters = {}) => {
  const [filters, setFilters] = useState({
    page: 1,
    page_size: 20,
    sort_by: 'created_at',
    sort_order: 'desc',
    ...initialFilters,
  });

  const [isSubmitting, setIsSubmitting] = useState(false);

  // SWR key function
  const getKey = useCallback((filters) => {
    return ['trackers', filters];
  }, []);

  // SWR fetcher function
  const fetcher = useCallback(async ([, filters]) => {
    const response = await trackerAPI.getAll(filters);
    return response.data;
  }, []);

  // Use SWR for data fetching with automatic revalidation
  const {
    data,
    error,
    mutate,
    isLoading,
    isValidating,
  } = useSWR(getKey(filters), fetcher, {
    refreshInterval: 30000, // Refresh every 30 seconds
    revalidateOnFocus: true,
    revalidateOnReconnect: true,
    dedupingInterval: 5000, // Dedupe requests within 5 seconds
  });

  // Extract data properties
  const trackers = data?.trackers || [];
  const pagination = data?.pagination || {
    total: 0,
    page: 1,
    page_size: 20,
    total_pages: 0,
  };

  // Add tracker function
  const addTracker = useCallback(async (url, name, userId = null) => {
    try {
      setIsSubmitting(true);
      const response = await trackerAPI.add(url, name, userId);
      
      // Optimistically update the cache
      await mutate();
      
      return response.data;
    } catch (error) {
      console.error('Failed to add tracker:', error);
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  }, [mutate]);

  // Update tracker function
  const updateTracker = useCallback(async (id, data) => {
    try {
      setIsSubmitting(true);
      const response = await trackerAPI.update(id, data);
      
      // Optimistically update the cache
      await mutate();
      
      return response.data;
    } catch (error) {
      console.error('Failed to update tracker:', error);
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  }, [mutate]);

  // Delete tracker function
  const deleteTracker = useCallback(async (id) => {
    try {
      setIsSubmitting(true);
      await trackerAPI.delete(id);
      
      // Optimistically update the cache
      await mutate();
      
      return true;
    } catch (error) {
      console.error('Failed to delete tracker:', error);
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  }, [mutate]);

  // Bulk add trackers function
  const bulkAddTrackers = useCallback(async (trackers) => {
    try {
      setIsSubmitting(true);
      const response = await trackerAPI.bulkAdd(trackers);
      
      // Optimistically update the cache
      await mutate();
      
      return response.data;
    } catch (error) {
      console.error('Failed to bulk add trackers:', error);
      throw error;
    } finally {
      setIsSubmitting(false);
    }
  }, [mutate]);

  // Update filters function
  const updateFilters = useCallback((newFilters) => {
    setFilters(prev => ({
      ...prev,
      ...newFilters,
      page: newFilters.page || 1, // Reset to page 1 unless explicitly specified
    }));
  }, []);

  // Reset filters function
  const resetFilters = useCallback(() => {
    setFilters({
      page: 1,
      page_size: 20,
      sort_by: 'created_at',
      sort_order: 'desc',
    });
  }, []);

  // Refresh data function
  const refresh = useCallback(() => {
    return mutate();
  }, [mutate]);

  return {
    // Data
    trackers,
    pagination,
    filters,
    
    // Loading states
    isLoading: isLoading || isValidating,
    isSubmitting,
    error,
    
    // Actions
    addTracker,
    updateTracker,
    deleteTracker,
    bulkAddTrackers,
    updateFilters,
    resetFilters,
    refresh,
  };
};

export default useTrackers; 