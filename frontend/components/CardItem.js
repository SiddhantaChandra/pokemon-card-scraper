import { useState } from 'react';
import Image from 'next/image';
import { apiHelpers } from '../lib/api';

export default function CardItem({ card }) {
  const [imageError, setImageError] = useState(false);
  const [imageLoading, setImageLoading] = useState(true);

  const stockStatus = apiHelpers.getStockStatus(card.stock);
  const externalUrl = apiHelpers.getExternalCardUrl(card);
  const imageUrl = apiHelpers.getCardImageUrl(card);

  const handleImageLoad = () => {
    setImageLoading(false);
  };

  const handleImageError = () => {
    setImageError(true);
    setImageLoading(false);
  };

  const handleCardClick = () => {
    if (externalUrl) {
      window.open(externalUrl, '_blank', 'noopener,noreferrer');
    }
  };

  return (
    <div className="group bg-white rounded-lg shadow-md hover:shadow-lg transition-all duration-200 overflow-hidden border border-gray-200 hover:border-gray-300">
      {/* Image Container */}
      <div className="relative aspect-[3/4] bg-gray-100 overflow-hidden">
        {imageLoading && (
          <div className="absolute inset-0 flex items-center justify-center z-10">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        )}
        
        {!imageError ? (
          <Image
            src={imageUrl}
            alt={card.name || 'Pokemon Card'}
            fill
            className={`object-cover transition-transform duration-200 group-hover:scale-105 ${
              imageLoading ? 'opacity-0' : 'opacity-100'
            }`}
            onLoad={handleImageLoad}
            onError={handleImageError}
            loading="lazy"
            unoptimized={true}
            crossOrigin="anonymous"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center bg-gray-200">
            <div className="text-center text-gray-500">
              <svg className="mx-auto h-12 w-12 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <p className="text-xs">No Image</p>
            </div>
          </div>
        )}

        {/* Stock Status Badge */}
        <div className="absolute top-2 right-2 z-20">
          <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${stockStatus.color} ${stockStatus.bgColor}`}>
            {stockStatus.text}
          </span>
        </div>

        {/* Condition Badge */}
        {card.condition && (
          <div className="absolute bottom-2 right-2 z-20">
            <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
              card.condition === 'Perfect' ? 'bg-green-100 text-green-800' :
              card.condition.startsWith('A') ? 'bg-blue-100 text-blue-800' :
              card.condition.startsWith('B') ? 'bg-yellow-100 text-yellow-800' :
              'bg-red-100 text-red-800'
            }`}>
              {card.condition}
            </span>
          </div>
        )}

        {/* External Link Indicator */}
        {externalUrl && (
          <div className="absolute top-2 left-2 opacity-0 group-hover:opacity-100 transition-opacity duration-200 z-20">
            <div className="bg-black bg-opacity-75 text-white p-1 rounded">
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
              </svg>
            </div>
          </div>
        )}
      </div>

      {/* Card Content */}
      <div className="p-4">
        {/* Card Name */}
        <h3 className="font-semibold text-gray-900 text-sm mb-2 line-clamp-2 min-h-[2.5rem]">
          {card.name}
        </h3>

        {/* Japanese Name */}
        {card.name_jp && card.name_jp !== card.name && (
          <p className="text-xs text-gray-500 mb-2 line-clamp-1">
            {card.name_jp}
          </p>
        )}

        {/* Set and Rarity */}
        <div className="flex items-center justify-between mb-3 text-xs text-gray-600">
          {card.set_name && (
            <span className="bg-gray-100 px-2 py-1 rounded truncate max-w-[60%]">
              {card.set_name}
            </span>
          )}
          {card.rarity && (
            <span className="bg-yellow-100 text-yellow-800 px-2 py-1 rounded">
              {card.rarity}
            </span>
          )}
        </div>

        {/* Price */}
        <div className="flex items-center justify-between">
          <div className="text-lg font-bold text-gray-900">
            {apiHelpers.formatPrice(card.price)}
          </div>
          
          {/* Click to view button */}
          {externalUrl && (
            <button
              onClick={handleCardClick}
              className="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-xs font-medium transition-colors duration-200 flex items-center space-x-1"
            >
              <span>View</span>
              <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
              </svg>
            </button>
          )}
        </div>

        {/* Last Updated */}
        {card.updated_at && (
          <div className="mt-2 text-xs text-gray-400">
            Updated {apiHelpers.formatDate(card.updated_at)}
          </div>
        )}
      </div>
    </div>
  );
} 