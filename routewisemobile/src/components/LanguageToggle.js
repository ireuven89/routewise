import React from 'react';
import { TouchableOpacity, Text, StyleSheet } from 'react-native';
import { useLanguage } from '../i18n/LanguageContext';
import { colors, theme } from '../theme/colors';

// Shows the *other* language's name (עברית while in English, English while in Hebrew).
const LanguageToggle = ({ style, textStyle }) => {
  const { t, toggleLanguage } = useLanguage();
  return (
    <TouchableOpacity
      onPress={toggleLanguage}
      style={[styles.button, style]}
      accessibilityRole="button"
      testID="language-toggle"
    >
      <Text style={[styles.text, textStyle]}>🌐 {t('common.language')}</Text>
    </TouchableOpacity>
  );
};

const styles = StyleSheet.create({
  button: {
    paddingHorizontal: theme.spacing.sm,
    paddingVertical: theme.spacing.xs,
    borderRadius: theme.borderRadius.full,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.4)',
  },
  text: {
    color: colors.textWhite,
    fontSize: theme.fontSize.sm,
    fontWeight: '600',
  },
});

export default LanguageToggle;
