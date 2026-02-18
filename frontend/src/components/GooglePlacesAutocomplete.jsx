import React, { useEffect, useRef } from 'react';

/**
 * Google Places Autocomplete Component
 * Uses the new PlaceAutocompleteElement (recommended by Google as of March 2025)
 * Uncontrolled component - the web component manages its own state
 */
const GooglePlacesAutocomplete = ({
  onChange,
  placeholder = 'Enter address',
  required = false,
  apiKey,
}) => {
  const containerRef = useRef(null);
  const autocompleteRef = useRef(null);

  useEffect(() => {
    if (!apiKey || !window.google || !window.google.maps) {
      console.error('Google Maps not loaded or API key missing');
      return;
    }

    if (!window.google.maps.places || !window.google.maps.places.PlaceAutocompleteElement) {
      console.error('PlaceAutocompleteElement not available - make sure Places API (New) is enabled');
      return;
    }

    try {
      console.log('✅ Initializing PlaceAutocompleteElement...');

      // Create the autocomplete element
      const autocomplete = new window.google.maps.places.PlaceAutocompleteElement();

      // Set placeholder
      autocomplete.setAttribute('placeholder', placeholder);

      // Listen for place selection using correct event name 'gmp-select'
      autocomplete.addEventListener('gmp-select', async ({ placePrediction }) => {
        console.log('🎯 Place selected event fired!');

        if (!placePrediction) {
          console.error('❌ No placePrediction in event');
          return;
        }

        try {
          // Convert placePrediction to Place object
          const place = placePrediction.toPlace();

          // Fetch place details
          await place.fetchFields({
            fields: ['displayName', 'formattedAddress', 'location', 'addressComponents', 'id'],
          });

          console.log('📍 Place details:', {
            displayName: place.displayName,
            formattedAddress: place.formattedAddress,
            location: place.location,
          });

          const location = place.location;
          if (!location) {
            console.error('❌ No location data');
            return;
          }

          // Extract address components
          const addressComponents = {};
          if (place.addressComponents) {
            place.addressComponents.forEach((component) => {
              const types = component.types || [];
              const componentType = types[0];
              switch (componentType) {
                case 'street_number':
                  addressComponents.street_number = component.longText;
                  break;
                case 'route':
                  addressComponents.street = component.longText;
                  break;
                case 'locality':
                  addressComponents.city = component.longText;
                  break;
                case 'administrative_area_level_1':
                  addressComponents.state = component.shortText;
                  break;
                case 'country':
                  addressComponents.country = component.longText;
                  addressComponents.country_code = component.shortText;
                  break;
                case 'postal_code':
                  addressComponents.postal_code = component.longText;
                  break;
                default:
                  break;
              }
            });
          }

          // Build result
          const addressText = place.formattedAddress || place.displayName || '';
          const result = {
            address: addressText,
            latitude: location.lat(),
            longitude: location.lng(),
            placeId: place.id,
            formattedAddress: place.formattedAddress || addressText,
            components: addressComponents,
          };

          console.log('✅ Result:', result);

          if (onChange) {
            onChange(result);
          }
        } catch (error) {
          console.error('❌ Error processing place selection:', error);
        }
      });

      // Clear container and append
      if (containerRef.current) {
        containerRef.current.innerHTML = '';
        containerRef.current.appendChild(autocomplete);
        autocompleteRef.current = autocomplete;
        console.log('✅ Autocomplete element added to DOM');
      }

      // Cleanup
      return () => {
        if (containerRef.current) {
          containerRef.current.innerHTML = '';
        }
      };
    } catch (error) {
      console.error('❌ Failed to initialize PlaceAutocompleteElement:', error);
    }
  }, [apiKey, placeholder]); // Removed onChange from dependencies to prevent re-initialization

  return (
    <div
      ref={containerRef}
      className="mt-1 block w-full"
      style={{ minHeight: '42px' }}
    />
  );
};

export default GooglePlacesAutocomplete;
