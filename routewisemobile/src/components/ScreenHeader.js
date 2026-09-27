import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { useRTL } from '../i18n/LanguageContext';
import LanguageToggle from './LanguageToggle';
import { colors, theme } from '../theme/colors';

// Navy header used by the customer screens: optional back arrow, title, language toggle, optional right action.
const ScreenHeader = ({ title, subtitle, onBack, action }) => {
  const rtl = useRTL();
  return (
    <View style={styles.header}>
      <View style={[styles.topRow, rtl.row]}>
        {onBack ? (
          <TouchableOpacity onPress={onBack} testID="header-back" hitSlop={12}>
            <Text style={styles.back}>{rtl.arrowBack}</Text>
          </TouchableOpacity>
        ) : (
          <Text style={styles.brand}>RouteWise</Text>
        )}
        <View style={[styles.actions, rtl.row]}>
          {action}
          <LanguageToggle />
        </View>
      </View>
      {title ? <Text style={[styles.title, rtl.text]}>{title}</Text> : null}
      {subtitle ? <Text style={[styles.subtitle, rtl.text]}>{subtitle}</Text> : null}
    </View>
  );
};

const styles = StyleSheet.create({
  header: {
    backgroundColor: colors.primary,
    paddingTop: 56,
    paddingBottom: theme.spacing.lg,
    paddingHorizontal: theme.spacing.lg,
  },
  topRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: theme.spacing.md,
  },
  actions: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing.sm,
  },
  brand: {
    color: colors.textWhite,
    fontSize: theme.fontSize.lg,
    fontWeight: 'bold',
  },
  back: {
    color: colors.textWhite,
    fontSize: theme.fontSize.xl,
    fontWeight: 'bold',
  },
  title: {
    color: colors.textWhite,
    fontSize: theme.fontSize.xl,
    fontWeight: 'bold',
  },
  subtitle: {
    color: colors.textWhite,
    opacity: 0.85,
    fontSize: theme.fontSize.md,
    marginTop: theme.spacing.xs,
  },
});

export default ScreenHeader;
