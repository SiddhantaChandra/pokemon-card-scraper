import CardItem from './CardItem';

export default function CardGrid({ 
  cards = [], 
  isLoading = false, 
  isEmpty = false, 
  error = null,
  className = '' 
}) {
  // Loading state
  if (isLoading) {
    return (
      <div className={`grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4 ${className}`}>
        {Array.from({ length: 20 }).map((_, index) => (
          <CardSkeleton key={index} />
        ))}
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <div className="text-center">
          <svg className="mx-auto h-12 w-12 text-red-400 dark:text-red-500 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 15.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">Error Loading Cards</h3>
          <p className="text-gray-500 dark:text-gray-400 mb-4">{error.message || 'Something went wrong while loading the cards.'}</p>
          <button 
            onClick={() => window.location.reload()}
            className="bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 text-white px-4 py-2 rounded-md font-medium transition-colors duration-200"
          >
            Try Again
          </button>
        </div>
      </div>
    );
  }

  // Empty state
  if (isEmpty) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <div className="text-center">
          <svg className="mx-auto h-12 w-12 text-gray-400 dark:text-gray-500 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">No Cards Found</h3>
          <p className="text-gray-500 dark:text-gray-400 mb-4">
            We couldn't find any cards matching your search criteria.
          </p>
          <p className="text-sm text-gray-400 dark:text-gray-500">
            Try adjusting your filters or search terms.
          </p>
        </div>
      </div>
    );
  }

  // Cards grid
  return (
    <div className={`grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4 ${className}`}>
      {cards.map((card) => (
        <CardItem 
          key={card.id || `${card.name}-${card.price}`} 
          card={card} 
        />
      ))}
    </div>
  );
}

// Skeleton loader component for loading state
function CardSkeleton() {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md overflow-hidden border border-gray-200 dark:border-gray-700 animate-pulse">
      {/* Image skeleton */}
      <div className="aspect-[3/4] bg-gray-300 dark:bg-gray-600"></div>
      
      {/* Content skeleton */}
      <div className="p-4">
        {/* Title skeleton */}
        <div className="h-4 bg-gray-300 dark:bg-gray-600 rounded mb-2"></div>
        <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded mb-3 w-3/4"></div>
        
        {/* Set and rarity skeleton */}
        <div className="flex items-center justify-between mb-3">
          <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-16"></div>
          <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-12"></div>
        </div>
        
        {/* Price skeleton */}
        <div className="flex items-center justify-between">
          <div className="h-6 bg-gray-300 dark:bg-gray-600 rounded w-20"></div>
          <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-12"></div>
        </div>
        
        {/* Date skeleton */}
        <div className="mt-2 h-3 bg-gray-200 dark:bg-gray-700 rounded w-24"></div>
      </div>
    </div>
  );
} 