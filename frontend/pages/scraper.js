import { useState } from 'react';
import Head from 'next/head';
import Image from 'next/image';
import Link from 'next/link';
import { useDarkMode } from '../lib/darkModeContext';
import ScraperControls from '../components/ScraperControls';
import ScrapeStatus from '../components/ScrapeStatus';
import { useCards } from '../hooks/useCards';

export default function Scraper() {
  const { isDarkMode, toggleDarkMode } = useDarkMode();
  const { totalItems, refresh, isValidating } = useCards('', {});

  return (
    <>
      <Head>
        <title>SCRAPER 9000 X MAX - Scraper</title>
        <meta name="description" content="Control and monitor Pokemon card scraping operations" />
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
                <Link href="/tracker" className="flex items-center space-x-2 py-4 px-1 border-b-2 border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600 font-medium text-sm transition-colors">
                  <span className="text-lg">📍</span>
                  <span>Tracker</span>
                </Link>
                <div className="flex items-center space-x-2 py-4 px-1 border-b-2 border-blue-500 text-blue-600 dark:text-blue-400 font-medium text-sm">
                  <span className="text-lg">⚙️</span>
                  <span>Scraper</span>
                </div>
                
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
            <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-6">Scraper Control & Status</h2>
              
              {/* Scraper Controls */}
              <div className="mb-8">
                <ScraperControls />
              </div>

              {/* Scraper Status */}
              <div>
                <ScrapeStatus />
              </div>
            </div>
          </div>
        </main>
      </div>
    </>
  );
} 