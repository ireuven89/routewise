import React from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { useLanguage, useRTL } from '../../i18n/LanguageContext';
import { useMode } from '../../navigation/ModeContext';
import LanguageToggle from '../../components/LanguageToggle';
import { colors, theme } from '../../theme/colors';

const RoleSelectScreen = () => {
  const { t } = useLanguage();
  const rtl = useRTL();
  const { setMode } = useMode();

  const RoleCard = ({ mode, icon, title, sub }) => (
    <TouchableOpacity
      style={[styles.card, rtl.row]}
      onPress={() => setMode(mode)}
      testID={`role-${mode}`}
      accessibilityRole="button"
    >
      <Text style={styles.icon}>{icon}</Text>
      <View style={styles.cardBody}>
        <Text style={[styles.cardTitle, rtl.text]}>{title}</Text>
        <Text style={[styles.cardSub, rtl.text]}>{sub}</Text>
      </View>
    </TouchableOpacity>
  );

  return (
    <View style={styles.container}>
      <View style={[styles.topRow, rtl.row]}>
        <View />
        <LanguageToggle />
      </View>

      <View style={styles.hero}>
        <Text style={styles.logo}>RouteWise</Text>
        <Text style={styles.tagline}>{t('roleSelect.tagline')}</Text>
      </View>

      <View style={styles.cards}>
        <RoleCard
          mode="customer"
          icon="🏠"
          title={t('roleSelect.customerTitle')}
          sub={t('roleSelect.customerSub')}
        />
        <RoleCard
          mode="technician"
          icon="🔧"
          title={t('roleSelect.technicianTitle')}
          sub={t('roleSelect.technicianSub')}
        />
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.primary,
    paddingHorizontal: theme.spacing.lg,
    paddingTop: 56,
  },
  topRow: { flexDirection: 'row', justifyContent: 'space-between' },
  hero: { alignItems: 'center', marginTop: theme.spacing.xl * 2, marginBottom: theme.spacing.xl },
  logo: { color: colors.textWhite, fontSize: theme.fontSize.xxl, fontWeight: 'bold' },
  tagline: { color: colors.textWhite, opacity: 0.85, fontSize: theme.fontSize.lg, marginTop: theme.spacing.sm },
  cards: { gap: theme.spacing.md },
  card: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.backgroundWhite,
    borderRadius: theme.borderRadius.lg,
    padding: theme.spacing.lg,
    gap: theme.spacing.md,
    ...theme.shadow.md,
  },
  icon: { fontSize: 36 },
  cardBody: { flex: 1 },
  cardTitle: { fontSize: theme.fontSize.lg, fontWeight: 'bold', color: colors.text },
  cardSub: { fontSize: theme.fontSize.sm, color: colors.textSecondary, marginTop: 4 },
});

export default RoleSelectScreen;
