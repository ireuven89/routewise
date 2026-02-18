// Singleton promise to ensure Google Maps script is loaded only once
let googleMapsPromise = null;

/**
 * Loads the Google Maps JavaScript API script
 * Uses singleton pattern to ensure script is only loaded once
 * @param {string} apiKey - Google Maps API key
 * @returns {Promise<void>} Resolves when the script is loaded
 */
export const loadGoogleMapsScript = (apiKey) => {
  // Return existing promise if script is already loading or loaded
  if (googleMapsPromise) {
    return googleMapsPromise;
  }

  // Check if script is already loaded
  if (window.google && window.google.maps) {
    return Promise.resolve();
  }

  // Create new promise to load the script
  googleMapsPromise = new Promise((resolve, reject) => {
    try {
      // Create script element
      const script = document.createElement('script');
      script.src = `https://maps.googleapis.com/maps/api/js?key=${apiKey}&libraries=places`;
      script.async = true;
      script.defer = true;

      // Handle successful load
      script.addEventListener('load', () => {
        resolve();
      });

      // Handle errors
      script.addEventListener('error', (error) => {
        googleMapsPromise = null; // Reset so it can be retried
        reject(new Error('Failed to load Google Maps script'));
      });

      // Add script to document
      document.head.appendChild(script);
    } catch (error) {
      googleMapsPromise = null; // Reset so it can be retried
      reject(error);
    }
  });

  return googleMapsPromise;
};

/**
 * Checks if Google Maps API is loaded
 * @returns {boolean} True if Google Maps is available
 */
export const isGoogleMapsLoaded = () => {
  return !!(window.google && window.google.maps);
};
