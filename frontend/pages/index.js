import { useState, useEffect } from 'react';
import Head from 'next/head';
import { useSearch } from '../hooks/useSearch';
import { useCards } from '../hooks/useCards';
import SearchBar from '../components/SearchBar';
import CardGrid from '../components/CardGrid';
import ScrapeStatus from '../components/ScrapeStatus';
import Pagination from '../components/Pagination';

export default function Home() {
  const [showScrapeStatus, setShowScrapeStatus] = useState(false);
  
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
        const sort = urlParams.get('sort') || 'date_desc';

        setSearchQuery(query);
        updateFilters({
          page,
          page_size: pageSize,
          min_price: minPrice ? parseFloat(minPrice) : undefined,
          max_price: maxPrice ? parseFloat(maxPrice) : undefined,
          in_stock: inStock,
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
      if (filters.sort !== 'date_desc') params.set('sort', filters.sort);

      const newUrl = params.toString() ? `?${params.toString()}` : '/';
      window.history.replaceState({}, '', newUrl);
    }
  }, [query, filters]);

  return (
    <>
      <Head>
        <title>Pokemon Card Scraper - Find the Best Deals</title>
        <meta name="description" content="Search and find Pokemon cards from torecacamp-pokemon.com with real-time pricing and stock information." />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/favicon.ico" />
      </Head>

      <div className="min-h-screen bg-gray-50">
        {/* Header */}
        <header className="bg-white shadow-sm border-b border-gray-200">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div className="flex items-center justify-between h-16">
              <div className="flex items-center space-x-4">
                <h1 className="text-2xl font-bold text-gray-900">
                  Pokemon Card Scraper
                </h1>
                <div className="hidden sm:block">
                  <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                    {totalItems.toLocaleString()} cards
                  </span>
                </div>
              </div>

              <div className="flex items-center space-x-4">
                {/* Refresh Button */}
                <button
                  onClick={refresh}
                  disabled={isLoading}
                  className="p-2 text-gray-400 hover:text-gray-600 transition-colors disabled:opacity-50"
                  title="Refresh data"
                >
                  <svg className={`h-5 w-5 ${isValidating ? 'animate-spin' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                </button>

                {/* Scrape Status Toggle */}
                <button
                  onClick={() => setShowScrapeStatus(!showScrapeStatus)}
                  className="flex items-center space-x-2 px-3 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors"
                >
                  <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                  </svg>
                  <span className="hidden sm:inline">Scraper</span>
                </button>
              </div>
            </div>
          </div>
        </header>

        {/* Main Content */}
        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="space-y-6">
            {/* Scrape Status (Collapsible) */}
            {showScrapeStatus && (
              <ScrapeStatus />
            )}

            {/* Search Bar */}
            <SearchBar
              query={query}
              onQueryChange={setSearchQuery}
              filters={filters}
              onFiltersChange={updateFilters}
            />

            {/* Active Filters Summary */}
            {hasActiveFilters() && (
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2 text-sm text-blue-800">
                    <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 100 4m0-4v2m0-6V4" />
                    </svg>
                    <span className="font-medium">Active filters:</span>
                    <div className="flex items-center space-x-2">
                                             {query && (
                         <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                           Search: &ldquo;{query}&rdquo;
                         </span>
                       )}
                      {filters.min_price && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                          Min: ¥{filters.min_price}
                        </span>
                      )}
                      {filters.max_price && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                          Max: ¥{filters.max_price}
                        </span>
                      )}
                      {filters.in_stock && (
                        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                          In Stock Only
                        </span>
                      )}
                    </div>
                  </div>
                  <button
                    onClick={clearSearch}
                    className="text-sm text-blue-600 hover:text-blue-800 font-medium"
                  >
                    Clear All
                  </button>
                </div>
              </div>
            )}

            {/* Results Summary */}
            {!isLoading && !error && (
              <div className="flex items-center justify-between text-sm text-gray-600">
                <div>
                  {totalItems > 0 ? (
                                         <>
                       Found <span className="font-medium text-gray-900">{totalItems.toLocaleString()}</span> cards
                       {query && <span> matching &ldquo;{query}&rdquo;</span>}
                     </>
                  ) : (
                    <span>No cards found</span>
                  )}
                </div>
                {totalPages > 1 && (
                  <div>
                    Page <span className="font-medium text-gray-900">{currentPage}</span> of{' '}
                    <span className="font-medium text-gray-900">{totalPages}</span>
                  </div>
                )}
              </div>
            )}

            {/* Cards Grid */}
            <CardGrid
              cards={cards}
              isLoading={isLoading}
              isEmpty={isEmpty}
              error={error}
            />

            {/* Pagination */}
            {totalPages > 1 && (
              <Pagination
                currentPage={currentPage}
                totalPages={totalPages}
                totalItems={totalItems}
                pageSize={pageSize}
                onPageChange={goToPage}
                className="mt-8"
              />
            )}

            {/* Footer Info */}
            <div className="text-center text-sm text-gray-500 py-8 border-t border-gray-200">
              <p>
                Data scraped from{' '}
                <a
                  href="https://torecacamp-pokemon.com"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-600 hover:text-blue-800"
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
