import { StyleSheet } from 'react-native';
import { colors, theme } from '../../theme/colors';

export const SERVICE_TYPES = [
  { value: 'hvac', icon: '❄️' },
  { value: 'plumbing', icon: '💧' },
  { value: 'electrical', icon: '⚡' },
];

// Shared look for the customer flow (navy header, white cards, orange primary button — same as the web).
export const shared = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background },
  scroll: { padding: theme.spacing.md, paddingBottom: theme.spacing.xl },
  card: {
    backgroundColor: colors.cardBackground,
    borderRadius: theme.borderRadius.lg,
    padding: theme.spacing.md,
    marginBottom: theme.spacing.md,
    ...theme.shadow.sm,
  },
  label: {
    fontSize: theme.fontSize.sm,
    fontWeight: '600',
    color: colors.text,
    marginBottom: theme.spacing.xs,
  },
  input: {
    backgroundColor: colors.inputBg,
    borderRadius: theme.borderRadius.sm,
    padding: theme.spacing.md,
    fontSize: theme.fontSize.md,
    color: colors.text,
    minHeight: 48,
  },
  field: { marginBottom: theme.spacing.md },
  hint: { fontSize: theme.fontSize.sm, color: colors.textSecondary, marginTop: theme.spacing.xs },
  primaryButton: {
    backgroundColor: colors.accent,
    borderRadius: theme.borderRadius.md,
    padding: theme.spacing.md,
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 50,
  },
  primaryButtonText: { color: colors.textWhite, fontSize: theme.fontSize.md, fontWeight: 'bold' },
  secondaryButton: {
    borderRadius: theme.borderRadius.md,
    padding: theme.spacing.md,
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 48,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundWhite,
    marginTop: theme.spacing.sm,
  },
  secondaryButtonText: { color: colors.primary, fontSize: theme.fontSize.md, fontWeight: '600' },
  disabled: { opacity: 0.5 },
  chipRow: { flexDirection: 'row', flexWrap: 'wrap', gap: theme.spacing.sm },
  chip: {
    paddingHorizontal: theme.spacing.md,
    paddingVertical: theme.spacing.sm,
    borderRadius: theme.borderRadius.full,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.backgroundWhite,
  },
  chipActive: { backgroundColor: colors.primary, borderColor: colors.primary },
  chipText: { color: colors.text, fontSize: theme.fontSize.sm, fontWeight: '600' },
  chipTextActive: { color: colors.textWhite },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: theme.spacing.lg },
  muted: { color: colors.textSecondary, fontSize: theme.fontSize.md, textAlign: 'center' },
  errorText: { color: colors.error, fontSize: theme.fontSize.md, textAlign: 'center' },
  badge: {
    paddingHorizontal: theme.spacing.sm,
    paddingVertical: 4,
    borderRadius: theme.borderRadius.full,
    alignSelf: 'flex-start',
  },
  badgeText: { color: colors.textWhite, fontSize: theme.fontSize.xs, fontWeight: '700' },
});

export const requestStatusColor = (status) => {
  switch (status) {
    case 'awarded':
      return colors.success;
    case 'cancelled':
      return colors.textSecondary;
    default:
      return colors.warning;
  }
};

export const requestStatusKey = (status) => {
  switch (status) {
    case 'awarded':
      return 'requestTracking.statusAwarded';
    case 'cancelled':
      return 'requestTracking.statusCancelled';
    default:
      return 'requestTracking.statusOpen';
  }
};
