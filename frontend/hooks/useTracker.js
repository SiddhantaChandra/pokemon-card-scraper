import { useState, useEffect, useCallback } from 'react';
import api from '../lib/api';

export function useTracker() {
  const [trackers, setTrackers] = useState([]);
  const [stats, setStats] = useState(null);
  const [workerStatus, setWorkerStatus] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Clear error after some time
  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => setError(null), 5000);
      return () => clearTimeout(timer);
    }
  }, [error]);

  // Load initial data
  const loadTrackers = useCallback(async () => {
    try {
      setError(null);
      const data = await api.getTrackers();
      setTrackers(data.trackers || []);
      return data;
    } catch (err) {
      console.error('Failed to load trackers:', err);
      setError(err.message || 'Failed to load trackers');
      return { trackers: [] };
    }
  }, []);

  const loadStats = useCallback(async () => {
    try {
      const data = await api.getTrackerStats();
      setStats(data);
      return data;
    } catch (err) {
      console.error('Failed to load tracker stats:', err);
      return null;
    }
  }, []);

  const loadWorkerStatus = useCallback(async () => {
    try {
      const data = await api.getTrackerStatus();
      setWorkerStatus(data);
      return data;
    } catch (err) {
      console.error('Failed to load worker status:', err);
      return null;
    }
  }, []);

  // Add a new tracker
  const addTracker = useCallback(async (trackerData) => {
    if (!trackerData) {
      const errorMessage = 'No tracker data provided';
      setError(errorMessage);
      throw new Error(errorMessage);
    }

    setLoading(true);
    setError(null);
    
    try {
      console.log('Adding tracker with data:', trackerData);
      const response = await api.addTracker(trackerData);
      console.log('Add tracker response:', response);
      
      // Handle different response structures
      const newTracker = response?.tracker || response;
      
      if (newTracker) {
        setTrackers(prev => [newTracker, ...prev]);
        // Refresh stats
        loadStats();
        return newTracker;
      } else {
        throw new Error('Invalid response from server');
      }
    } catch (err) {
      console.error('Failed to add tracker:', err);
      const errorMessage = err.response?.data?.message || err.message || 'Failed to add tracker';
      setError(errorMessage);
      throw new Error(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [loadStats]);

  // Delete a tracker
  const deleteTracker = useCallback(async (trackerId) => {
    setLoading(true);
    setError(null);
    
    try {
      await api.deleteTracker(trackerId);
      setTrackers(prev => prev.filter(t => t.id !== trackerId));
      
      // Refresh stats
      loadStats();
    } catch (err) {
      console.error('Failed to delete tracker:', err);
      const errorMessage = err.message || 'Failed to delete tracker';
      setError(errorMessage);
      throw new Error(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [loadStats]);

  // Trigger manual check
  const checkNow = useCallback(async () => {
    setLoading(true);
    setError(null);
    
    try {
      const result = await api.checkTrackersNow();
      
      // Refresh data after check
      await Promise.all([
        loadTrackers(),
        loadStats(),
        loadWorkerStatus()
      ]);
      
      return result;
    } catch (err) {
      console.error('Failed to check trackers:', err);
      const errorMessage = err.message || 'Failed to check trackers';
      setError(errorMessage);
      throw new Error(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [loadTrackers, loadStats, loadWorkerStatus]);

  // Refresh all data
  const refreshData = useCallback(async () => {
    setLoading(true);
    setError(null);
    
    try {
      await Promise.all([
        loadTrackers(),
        loadStats(),
        loadWorkerStatus()
      ]);
    } catch (err) {
      console.error('Failed to refresh data:', err);
      setError(err.message || 'Failed to refresh data');
    } finally {
      setLoading(false);
    }
  }, [loadTrackers, loadStats, loadWorkerStatus]);

  // Auto-refresh data periodically
  useEffect(() => {
    // Initial load
    refreshData();
    
    // Set up auto-refresh every 30 seconds
    const interval = setInterval(() => {
      loadStats();
      loadWorkerStatus();
    }, 30000);
    
    // Set up trackers refresh every 5 minutes
    const trackersInterval = setInterval(() => {
      loadTrackers();
    }, 300000);
    
    return () => {
      clearInterval(interval);
      clearInterval(trackersInterval);
    };
  }, [refreshData, loadStats, loadWorkerStatus, loadTrackers]);

  return {
    trackers,
    stats,
    workerStatus,
    loading,
    error,
    addTracker,
    deleteTracker,
    checkNow,
    refreshData,
    loadTrackers,
    loadStats,
    loadWorkerStatus
  };
} 