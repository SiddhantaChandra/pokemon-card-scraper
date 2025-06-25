import { useState, useEffect } from 'react';
import Head from 'next/head';
import Image from 'next/image';
import Link from 'next/link';
import { useDarkMode } from '../lib/darkModeContext';
import TrackerForm from '../components/TrackerForm';
import TrackerList from '../components/TrackerList';
import TrackerStats from '../components/TrackerStats';
import { useTracker } from '../hooks/useTracker';
import { useCards } from '../hooks/useCards';
import { trackerAPI } from '../lib/api';

export default function Tracker() {
  const { isDarkMode, toggleDarkMode } = useDarkMode();
  const {
    trackers,
    stats,
    workerStatus,
    loading,
    error,
    addTracker,
    deleteTracker,
    checkNow,
    refreshData
  } = useTracker();
  
  const { totalItems, refresh, isValidating } = useCards('', {});

  const [showAddForm, setShowAddForm] = useState(true);
  const [testingNotification, setTestingNotification] = useState(false);

  const handleTestNotification = async () => {
    setTestingNotification(true);
    try {
      const response = await trackerAPI.testNotification();
      if (response.success) {
        alert('Discord notification test completed! Check your Discord channel.');
      } else {
        alert('Discord notification test failed: ' + (response.message || 'Unknown error'));
      }
    } catch (error) {
      console.error('Test notification error:', error);
      alert('Failed to send test notification: ' + error.message);
    } finally {
      setTestingNotification(false);
    }
  };

  return (
    <>
      <Head>
        <title>SCRAPER 9000 X MAX - Tracker</title>
        <meta name="description" content="Track Pokemon card availability and prices" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
      </Head>

      <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
        {/* Header */}
        <header className="bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex items-center justify-between h-16">
              <div className="flex items-center space-x-4">
                <Image src="/images/PikaMascot.webp" alt="SCRAPER 9000" width={48} height={48} />
                <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                  SCRAPER 6900
                </h1>
                
                <div className="hidden sm:block">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200">
                    {totalItems.toLocaleString()} cards
                  </span>
                </div>
              </div>

              {/* Navigation */}
              <div className="flex space-x-8 -mb-px">
                <Link href="/" className="flex items-center space-x-2 py-4 px-1 border-b-2 border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600 font-medium text-sm transition-colors">
                  <span className="text-lg">🃏</span>
                  <span>Cards</span>
                </Link>
                <div className="flex items-center space-x-2 py-4 px-1 border-b-2 border-blue-500 text-blue-600 dark:text-blue-400 font-medium text-sm">
                  <span className="text-lg">📍</span>
                  <span>Tracker</span>
                </div>
                <Link href="/scraper" className="flex items-center space-x-2 py-4 px-1 border-b-2 border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600 font-medium text-sm transition-colors">
                  <span className="text-lg">⚙️</span>
                  <span>Scraper</span>
                </Link>
                
                <div className="flex items-center space-x-4">
                  {/* Dark Mode Toggle */}
                  <button
                    onClick={toggleDarkMode}
                    className="p-2 text-gray-400 dark:text-gray-300 hover:text-gray-600 dark:hover:text-gray-100 transition-colors"
                    title={isDarkMode ? 'Switch to light mode' : 'Switch to dark mode'}
                  >
                    {isDarkMode ? (
                      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
                      </svg>
                    ) : (
                      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
                      </svg>
                    )}
                  </button>

                  {/* Refresh Button */}
                  <button
                    onClick={refresh}
                    disabled={isValidating}
                    className="p-2 text-gray-400 dark:text-gray-300 hover:text-gray-600 dark:hover:text-gray-100 transition-colors disabled:opacity-50"
                    title="Refresh data"
                  >
                    <svg className={`h-5 w-5 ${isValidating ? 'animate-spin' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </header>

        {/* Main Content */}
        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="space-y-6">
            {/* Statistics */}
            <TrackerStats stats={stats} workerStatus={workerStatus} />

            {/* Add Tracker Form */}
            {showAddForm && (
              <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6">
                <div className="flex justify-between items-center mb-4">
                  <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Add New Tracker</h2>
                  <button
                    onClick={() => setShowAddForm(false)}
                    className="text-sm text-gray-600 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300"
                  >
                    Hide Form
                  </button>
                </div>
                <TrackerForm onAdd={addTracker} loading={loading} />
              </div>
            )}

            {/* Show Add Form Button (when hidden) */}
            {!showAddForm && (
              <div>
                <button
                  onClick={() => setShowAddForm(true)}
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
                >
                  + Add New Tracker
                </button>
              </div>
            )}

            {/* Action Buttons */}
            <div className="flex flex-wrap gap-4">
              <button
                onClick={checkNow}
                disabled={loading}
                className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'Checking...' : 'Check Now'}
              </button>
              
              <button
                onClick={refreshData}
                disabled={loading}
                className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? 'Refreshing...' : 'Refresh'}
              </button>

              <button
                onClick={handleTestNotification}
                disabled={testingNotification}
                className="px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {testingNotification ? 'Testing...' : 'Test Discord'}
              </button>
            </div>

            {/* Error Display */}
            {error && (
              <div className="p-4 rounded-lg bg-red-100 dark:bg-red-900/20 border border-red-400 dark:border-red-800 text-red-700 dark:text-red-200">
                <div className="flex items-center">
                  <svg className="w-5 h-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                  </svg>
                  <span className="font-medium">Error: </span>
                  {error}
                </div>
              </div>
            )}

            {/* Trackers List */}
            <TrackerList
              trackers={trackers}
              onDelete={deleteTracker}
              loading={loading}
            />

            {/* Empty State */}
            {!loading && trackers.length === 0 && (
              <div className="text-center py-12 text-gray-500 dark:text-gray-400">
                <div className="mb-4">
                  <svg className="w-16 h-16 mx-auto opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v10a2 2 0 002 2h8a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                  </svg>
                </div>
                <h3 className="text-lg font-medium mb-2">No trackers yet</h3>
                <p>Add your first Pokemon card URL to start tracking availability and prices!</p>
              </div>
            )}
          </div>
        </main>
      </div>
    </>
  );
} 