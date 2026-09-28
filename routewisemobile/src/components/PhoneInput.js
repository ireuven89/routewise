import React, { useState } from 'react';
import { View, Text, TextInput, TouchableOpacity, StyleSheet } from 'react-native';
import { colors, theme } from '../theme/colors';

export const COUNTRY_CODES = [
  { code: '+1', flag: '🇺🇸', label: 'US/Canada' },
  { code: '+972', flag: '🇮🇱', label: 'Israel' },
  { code: '+44', flag: '🇬🇧', label: 'UK' },
  { code: '+61', flag: '🇦🇺', label: 'Australia' },
  { code: '+91', flag: '🇮🇳', label: 'India' },
  { code: '+49', flag: '🇩🇪', label: 'Germany' },
  { code: '+33', flag: '🇫🇷', label: 'France' },
  { code: '+52', flag: '🇲🇽', label: 'Mexico' },
];

// Country-code picker + number field. Phone numbers stay LTR even in Hebrew.
const PhoneInput = ({ countryCode, onCountryCodeChange, phoneNumber, onPhoneNumberChange, testID }) => {
  const [showPicker, setShowPicker] = useState(false);

  return (
    <View>
      <View style={styles.phoneRow}>
        <TouchableOpacity style={styles.countryCodeButton} onPress={() => setShowPicker(!showPicker)}>
          <Text style={styles.countryCodeText}>
            {COUNTRY_CODES.find((c) => c.code === countryCode)?.flag} {countryCode}
          </Text>
        </TouchableOpacity>

        <TextInput
          style={styles.phoneInput}
          placeholder="0501234567"
          value={phoneNumber}
          onChangeText={onPhoneNumberChange}
          keyboardType="phone-pad"
          autoCorrect={false}
          testID={testID}
        />
      </View>

      {showPicker && (
        <View style={styles.dropdown}>
          {COUNTRY_CODES.map((item) => (
            <TouchableOpacity
              key={item.code}
              style={[styles.item, countryCode === item.code && styles.itemActive]}
              onPress={() => {
                onCountryCodeChange(item.code);
                setShowPicker(false);
              }}
            >
              <Text style={styles.itemText}>
                {item.flag}  {item.code}  {item.label}
              </Text>
            </TouchableOpacity>
          ))}
        </View>
      )}
    </View>
  );
};

const styles = StyleSheet.create({
  phoneRow: {
    flexDirection: 'row',
    gap: 8,
  },
  countryCodeButton: {
    backgroundColor: colors.inputBg,
    borderRadius: theme.borderRadius.sm,
    paddingHorizontal: theme.spacing.sm,
    minHeight: 48,
    justifyContent: 'center',
    alignItems: 'center',
    minWidth: 80,
  },
  countryCodeText: {
    fontSize: theme.fontSize.md,
    color: colors.text,
    fontWeight: '600',
  },
  phoneInput: {
    flex: 1,
    backgroundColor: colors.inputBg,
    borderRadius: theme.borderRadius.sm,
    padding: theme.spacing.md,
    fontSize: theme.fontSize.md,
    color: colors.text,
    minHeight: 48,
    textAlign: 'left',
    writingDirection: 'ltr',
  },
  dropdown: {
    marginTop: 4,
    backgroundColor: colors.backgroundWhite,
    borderRadius: theme.borderRadius.sm,
    borderWidth: 1,
    borderColor: '#e0e0e0',
    maxHeight: 200,
    zIndex: 10,
  },
  item: {
    padding: theme.spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: '#f0f0f0',
  },
  itemActive: {
    backgroundColor: '#e8f0fe',
  },
  itemText: {
    fontSize: theme.fontSize.md,
    color: colors.text,
  },
});

export default PhoneInput;
