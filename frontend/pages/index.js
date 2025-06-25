import { useState, useEffect } from 'react';
import Image from 'next/image';
import Head from 'next/head';
import Link from 'next/link';
import { useSearch } from '../hooks/useSearch';
import { useCards } from '../hooks/useCards';
import { useTracker } from '../hooks/useTracker';
import { useDarkMode } from '../lib/darkModeContext';
import SearchBar from '../components/SearchBar';
import CardGrid from '../components/CardGrid';
import ScrapeStatus from '../components/ScrapeStatus';
import ScraperControls from '../components/ScraperControls';
import TrackerForm from '../components/TrackerForm';
import TrackerList from '../components/TrackerList';
import TrackerStats from '../components/TrackerStats';
import Pagination, { CompactPagination } from '../components/Pagination';

export default function Home() {
  const { isDarkMode, toggleDarkMode } = useDarkMode();
  
  // Search state management
  const {
    query,
    debouncedQuery,
    filters,
    setSearchQuery,
    updateFilter,
    updateFilters,
    clearSearch,
    goToPage,
    hasActiveFilters
  } = useSearch();

  // Cards data fetching
  const {
    cards,
    totalPages,
    totalItems,
    currentPage,
    pageSize,
    isLoading,
    isValidating,
    error,
    isEmpty,
    refresh
  } = useCards(debouncedQuery, filters);

  // Tracker data fetching
  const {
    trackers,
    stats: trackerStats,
    workerStatus,
    loading: trackerLoading,
    error: trackerError,
    addTracker,
    deleteTracker,
    checkNow,
    refreshData: refreshTrackerData
  } = useTracker();

  // Handle URL parameters on page load
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const urlParams = new URLSearchParams(window.location.search);
      if (urlParams.toString()) {
        // Load search state from URL parameters
        const query = urlParams.get('q') || '';
        const page = parseInt(urlParams.get('page')) || 1;
        const pageSize = parseInt(urlParams.get('page_size')) || 20;
        const minPrice = urlParams.get('min_price');
        const maxPrice = urlParams.get('max_price');
        const inStock = urlParams.get('in_stock') === 'true';
        const setName = urlParams.get('set') || '';
        const rarity = urlParams.get('rarity') || '';
        const conditions = urlParams.get('conditions')?.split(',').filter(c => c) || [];
        const sort = urlParams.get('sort') || 'date_desc';

        setSearchQuery(query);
        updateFilters({
          page,
          page_size: pageSize,
          min_price: minPrice ? parseFloat(minPrice) : undefined,
          max_price: maxPrice ? parseFloat(maxPrice) : undefined,
          in_stock: inStock,
          set: setName,
          rarity: rarity,
          conditions: conditions,
          sort
        });
      }
    }
  }, [setSearchQuery, updateFilters]);

  // Update URL when search state changes
  useEffect(() => {
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams();
      
      if (query) params.set('q', query);
      if (filters.page > 1) params.set('page', filters.page.toString());
      if (filters.page_size !== 20) params.set('page_size', filters.page_size.toString());
      if (filters.min_price) params.set('min_price', filters.min_price.toString());
      if (filters.max_price) params.set('max_price', filters.max_price.toString());
      if (filters.in_stock) params.set('in_stock', 'true');
      if (filters.set) params.set('set', filters.set);
      if (filters.rarity) params.set('rarity', filters.rarity);
      if (filters.conditions && filters.conditions.length > 0) {
        params.set('conditions', filters.conditions.join(','));
      }
      if (filters.sort !== 'date_desc') params.set('sort', filters.sort);

      const newUrl = params.toString() ? `?${params.toString()}` : '/';
      window.history.replaceState({}, '', newUrl);
    }
  }, [query, filters]);

  return (
    <>
      <Head>
        <title>SCRAPER 9000 X MAX</title>
        <meta name="description" content="Search and find Pokemon cards from torecacamp-pokemon.com with real-time pricing and stock information." />
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
                <div className="flex items-center space-x-2 py-4 px-1 border-b-2 border-blue-500 text-blue-600 dark:text-blue-400 font-medium text-sm">
                  <span className="text-lg">🃏</span>
                  <span>Cards</span>
                </div>
                <Link href="/tracker" className="flex items-center space-x-2 py-4 px-1 border-b-2 border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:border-gray-300 dark:hover:border-gray-600 font-medium text-sm transition-colors">
                  <span className="text-lg">📍</span>
                  <span>Tracker</span>
                </Link>
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
                    disabled={isLoading}
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
            {/* Search Bar */}
            <SearchBar
              query={query}
              onQueryChange={setSearchQuery}
              filters={filters}
              onFiltersChange={updateFilters}
            />

            {/* Active Filters Summary */}
            {hasActiveFilters() && (
              <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2 text-sm text-blue-800 dark:text-blue-200">
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4" />
                    </svg>
                    <span className="font-medium">Active filters:</span>
                    <div className="flex flex-wrap items-center gap-2">
                      {query && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-800 text-blue-800 dark:text-blue-200">
                          Search: &ldquo;{query}&rdquo;
                        </span>
                      )}
                      {filters.min_price && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-800 text-blue-800 dark:text-blue-200">
                          Min: ¥{filters.min_price}
                        </span>
                      )}
                      {filters.max_price && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-800 text-blue-800 dark:text-blue-200">
                          Max: ¥{filters.max_price}
                        </span>
                      )}
                      {filters.in_stock && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 dark:bg-green-800 text-green-800 dark:text-green-200">
                          In Stock Only
                        </span>
                      )}
                      {filters.set && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-purple-100 dark:bg-purple-800 text-purple-800 dark:text-purple-200">
                          Set: {filters.set}
                        </span>
                      )}
                      {filters.rarity && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-yellow-100 dark:bg-yellow-800 text-yellow-800 dark:text-yellow-200">
                          Rarity: {filters.rarity}
                        </span>
                      )}
                      {filters.conditions && filters.conditions.length > 0 && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-orange-100 dark:bg-orange-800 text-orange-800 dark:text-orange-200">
                          Conditions: {filters.conditions.join(', ')}
                        </span>
                      )}
                    </div>
                  </div>
                  <button
                    onClick={clearSearch}
                    className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 font-medium"
                  >
                    Clear All
                  </button>
                </div>
              </div>
            )}

            {/* Results Summary and Top Pagination */}
            {!isLoading && !error && (
              <div className="flex items-center justify-between text-sm text-gray-600 dark:text-gray-400">
                <div>
                  {totalItems > 0 ? (
                    <>
                      Found <span className="font-medium text-gray-900 dark:text-white">{totalItems.toLocaleString()}</span> cards
                      {query && <span> matching &ldquo;{query}&rdquo;</span>}
                    </>
                  ) : (
                    <span>No cards found</span>
                  )}
                </div>
                <div className="flex items-center space-x-4">
                  {totalPages > 1 && (
                    <div className="hidden sm:block">
                      Page <span className="font-medium text-gray-900 dark:text-white">{currentPage}</span> of{' '}
                      <span className="font-medium text-gray-900 dark:text-white">{totalPages}</span>
                    </div>
                  )}
                  {/* Top Compact Pagination */}
                  <CompactPagination
                    currentPage={currentPage}
                    totalPages={totalPages}
                    onPageChange={goToPage}
                  />
                </div>
              </div>
            )}

            {/* Cards Grid */}
            <div id="cards-grid-top">
              <CardGrid
                cards={cards}
                isLoading={isLoading}
                isEmpty={isEmpty}
                error={error}
              />
            </div>

            {/* Bottom Pagination with Scroll to Top */}
            {totalPages > 1 && (
              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                totalItems={totalItems}
                pageSize={pageSize}
                onPageChange={goToPage}
                className="mt-8"
                scrollToTop={true}
                scrollTarget="#cards-grid-top"
              />
            )}

            {/* Footer Info */}
            <div className="text-center text-sm text-gray-500 dark:text-gray-400 py-8 border-t border-gray-200 dark:border-gray-700">
              <p>
                Data scraped from{' '}
                <a
                  href="https://torecacamp-pokemon.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300"
                >
                  torecacamp-pokemon.com
                </a>
              </p>
              <p className="mt-1">
                Prices and availability are updated regularly through automated scraping.
              </p>
            </div>
          </div>
        </main>
      </div>
    </>
  );
}
