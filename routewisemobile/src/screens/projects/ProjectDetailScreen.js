import React, { useState, useEffect } from 'react';
import {
  View, Text, StyleSheet, ScrollView, TouchableOpacity, ActivityIndicator, Linking, Platform, Alert,
  Modal, TextInput,
} from 'react-native';
import { jobs } from '../../services/api';
import { colors, theme } from '../../theme/colors';
import { useLanguage, useRTL } from '../../i18n/LanguageContext';
import { notify } from '../../utils/confirm';

const ProjectDetailScreen = ({ navigation, route }) => {
  const { jobId } = route.params;
  const { t, locale } = useLanguage();
  const rtl = useRTL();
  const statusLabel = (status) => (status ? t(`status.${status.toLowerCase()}`) : t('status.pending'));
  const [job, setJob] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [busy, setBusy] = useState(false);
  const [showDecline, setShowDecline] = useState(false);
  const [declineReason, setDeclineReason] = useState('');

  useEffect(() => {
    loadJobDetails();
  }, [jobId]);

  const loadJobDetails = async () => {
    try {
      setError(null);
      const response = await jobs.getJobDetails(jobId);
      setJob(response);
    } catch (err) {
      console.error('Failed to load job details:', err);
      setError(t('projectDetail.loadFailed'));
    } finally {
      setLoading(false);
    }
  };

  // Accept / decline / start / complete all follow the same shape: call, then refresh or leave.
  const runAction = async (action, after = loadJobDetails) => {
    setBusy(true);
    try {
      await action();
      await after();
    } catch (err) {
      console.error('Job action failed:', err);
      notify(t('common.error'), err.response?.data?.error || t('projectDetail.actionFailed'));
    } finally {
      setBusy(false);
    }
  };

  const handleAccept = () => runAction(() => jobs.accept(jobId));

  // A declined job leaves this technician's list, so go back to it.
  const handleDecline = () => runAction(
    () => jobs.decline(jobId, declineReason.trim()),
    async () => {
      setShowDecline(false);
      navigation.goBack();
    },
  );

  const handleStatus = (status) => runAction(() => jobs.updateStatus(jobId, status));

  const handleNavigateToCustomer = async () => {
    const { customer } = job;

    if (!customer) {
      Alert.alert(t('common.error'), t('projectDetail.noCustomer'));
      return;
    }

    // Try to use coordinates first (more accurate)
    if (customer.latitude && customer.longitude) {
      const url = Platform.select({
        ios: `maps:0,0?q=${customer.latitude},${customer.longitude}`,
        android: `geo:0,0?q=${customer.latitude},${customer.longitude}`,
      });

      try {
        const supported = await Linking.canOpenURL(url);
        if (supported) {
          await Linking.openURL(url);
        } else {
          // Fallback to web URL
          const webUrl = `https://www.google.com/maps/search/?api=1&query=${customer.latitude},${customer.longitude}`;
          await Linking.openURL(webUrl);
        }
      } catch (err) {
        console.error('Failed to open maps:', err);
        Alert.alert(t('common.error'), t('projectDetail.mapsFailed'));
      }
    } else if (customer.address) {
      // Fallback to address if coordinates not available
      const encodedAddress = encodeURIComponent(customer.address);
      const url = Platform.select({
        ios: `maps:0,0?q=${encodedAddress}`,
        android: `geo:0,0?q=${encodedAddress}`,
      });

      try {
        const supported = await Linking.canOpenURL(url);
        if (supported) {
          await Linking.openURL(url);
        } else {
          const webUrl = `https://www.google.com/maps/search/?api=1&query=${encodedAddress}`;
          await Linking.openURL(webUrl);
        }
      } catch (err) {
        console.error('Failed to open maps:', err);
        Alert.alert(t('common.error'), t('projectDetail.mapsFailed'));
      }
    } else {
      Alert.alert(t('common.error'), t('projectDetail.noLocation'));
    }
  };

  const handleCallCustomer = async () => {
    const { customer } = job;
    if (!customer?.phone) {
      Alert.alert(t('common.error'), t('projectDetail.noPhone'));
      return;
    }

    const url = `tel:${customer.phone}`;
    try {
      const supported = await Linking.canOpenURL(url);
      if (supported) {
        await Linking.openURL(url);
      } else {
        Alert.alert(t('common.error'), t('projectDetail.cannotCall'));
      }
    } catch (err) {
      console.error('Failed to call customer:', err);
      Alert.alert(t('common.error'), t('projectDetail.callFailed'));
    }
  };

  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'completed':
        return colors.success;
      case 'in_progress':
        return colors.warning;
      case 'scheduled':
        return colors.info;
      default:
        return colors.textSecondary;
    }
  };

  const formatDate = (dateString) => {
    if (!dateString) return t('projects.notScheduled');
    const date = new Date(dateString);
    return date.toLocaleDateString(locale, {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  if (loading) {
    return (
      <View style={styles.container}>
        <View style={styles.centerContent}>
          <ActivityIndicator size="large" color={colors.primary} />
          <Text style={styles.loadingText}>{t('projectDetail.loading')}</Text>
        </View>
      </View>
    );
  }

  if (error || !job) {
    return (
      <View style={styles.container}>
        <View style={styles.centerContent}>
          <Text style={styles.errorText}>⚠️ {error || t('projectDetail.notFound')}</Text>
          <TouchableOpacity style={styles.retryButton} onPress={loadJobDetails}>
            <Text style={styles.retryButtonText}>{t('common.retry')}</Text>
          </TouchableOpacity>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <ScrollView contentContainerStyle={styles.scrollContent}>
        {/* Job Header */}
        <View style={[styles.header, rtl.row]}>
          <Text style={[styles.jobTitle, rtl.text]}>{job.title || t('projects.jobNumber', { id: job.id })}</Text>
          <View style={[styles.statusBadge, { backgroundColor: getStatusColor(job.status) }]}>
            <Text style={styles.statusText}>{statusLabel(job.status)}</Text>
          </View>
        </View>

        {/* Respond to the assignment, then work it */}
        {job.assignment_status === 'pending' ? (
          <View style={styles.offerCard} testID="assignment-offer">
            <Text style={[styles.offerTitle, rtl.text]}>🔔 {t('projectDetail.newAssignment')}</Text>
            <Text style={[styles.offerSub, rtl.text]}>{t('projectDetail.newAssignmentSub')}</Text>
            <View style={[styles.actionRow, rtl.row]}>
              <TouchableOpacity
                style={[styles.actionButton, styles.acceptButton, busy && styles.disabled]}
                onPress={handleAccept}
                disabled={busy}
                testID="accept-job"
              >
                {busy
                  ? <ActivityIndicator color={colors.textWhite} />
                  : <Text style={styles.actionButtonText}>✓ {t('projectDetail.accept')}</Text>}
              </TouchableOpacity>
              <TouchableOpacity
                style={[styles.actionButton, styles.declineButton, busy && styles.disabled]}
                onPress={() => setShowDecline(true)}
                disabled={busy}
                testID="decline-job"
              >
                <Text style={[styles.actionButtonText, { color: colors.error }]}>{t('projectDetail.decline')}</Text>
              </TouchableOpacity>
            </View>
          </View>
        ) : job.status === 'scheduled' || job.status === 'in_progress' ? (
          <TouchableOpacity
            style={[styles.actionButton, styles.primaryAction, busy && styles.disabled]}
            onPress={() => handleStatus(job.status === 'scheduled' ? 'in_progress' : 'completed')}
            disabled={busy}
            testID={job.status === 'scheduled' ? 'start-job' : 'complete-job'}
          >
            {busy
              ? <ActivityIndicator color={colors.textWhite} />
              : (
                <Text style={styles.actionButtonText}>
                  {job.status === 'scheduled' ? `▶ ${t('projectDetail.startJob')}` : `✓ ${t('projectDetail.completeJob')}`}
                </Text>
              )}
          </TouchableOpacity>
        ) : null}

        {/* Job Description */}
        {!!job.description && (
          <View style={styles.section}>
            <Text style={[styles.sectionTitle, rtl.text]}>{t('projectDetail.description')}</Text>
            <Text style={[styles.descriptionText, rtl.text]}>{job.description}</Text>
          </View>
        )}

        {/* Schedule */}
        <View style={styles.section}>
          <Text style={[styles.sectionTitle, rtl.text]}>{t('projectDetail.schedule')}</Text>
          <Text style={[styles.infoText, rtl.text]}>📅 {formatDate(job.scheduled_at || job.scheduled_date)}</Text>
        </View>

        {/* Customer Information */}
        {!!job.customer?.id && (
          <View style={styles.section}>
            <Text style={[styles.sectionTitle, rtl.text]}>{t('projectDetail.customer')}</Text>
            <View style={styles.customerCard}>
              <Text style={[styles.customerName, rtl.text]}>{job.customer.name}</Text>

              {!!job.customer.phone && (
                <TouchableOpacity style={[styles.infoRow, rtl.row]} onPress={handleCallCustomer}>
                  <Text style={styles.infoLabel}>📞 {t('projectDetail.phone')}</Text>
                  <Text style={[styles.infoValue, styles.linkText]}>{job.customer.phone}</Text>
                </TouchableOpacity>
              )}

              {!!job.customer.email && (
                <View style={[styles.infoRow, rtl.row]}>
                  <Text style={styles.infoLabel}>✉️ {t('projectDetail.email')}</Text>
                  <Text style={[styles.infoValue, rtl.text]}>{job.customer.email}</Text>
                </View>
              )}

              {!!job.customer.address && (
                <View style={[styles.infoRow, rtl.row]}>
                  <Text style={styles.infoLabel}>📍 {t('projectDetail.address')}</Text>
                  <Text style={[styles.infoValue, rtl.text]}>{job.customer.address}</Text>
                </View>
              )}

              {/* Navigation Button */}
              {!!(job.customer.address || (job.customer.latitude && job.customer.longitude)) && (
                <TouchableOpacity style={styles.navigateButton} onPress={handleNavigateToCustomer}>
                  <Text style={styles.navigateButtonText}>🗺️ {t('projectDetail.navigate')}</Text>
                </TouchableOpacity>
              )}
            </View>
          </View>
        )}

        {/* Notes */}
        {!!job.notes && (
          <View style={styles.section}>
            <Text style={[styles.sectionTitle, rtl.text]}>{t('projectDetail.notes')}</Text>
            <Text style={[styles.notesText, rtl.text]}>{job.notes}</Text>
          </View>
        )}
      </ScrollView>

      <Modal visible={showDecline} transparent animationType="fade" onRequestClose={() => setShowDecline(false)}>
        <View style={styles.modalBackdrop}>
          <View style={styles.modalCard}>
            <Text style={[styles.offerTitle, rtl.text]}>{t('projectDetail.declineTitle')}</Text>
            <Text style={[styles.modalLabel, rtl.text]}>{t('projectDetail.declineReasonLabel')}</Text>
            <TextInput
              style={[styles.modalInput, rtl.text]}
              value={declineReason}
              onChangeText={setDeclineReason}
              placeholder={t('projectDetail.declineReasonPlaceholder')}
              multiline
              testID="decline-reason"
            />
            <View style={[styles.actionRow, rtl.row]}>
              <TouchableOpacity
                style={[styles.actionButton, styles.declineConfirmButton, busy && styles.disabled]}
                onPress={handleDecline}
                disabled={busy}
                testID="confirm-decline"
              >
                <Text style={styles.actionButtonText}>{t('projectDetail.declineConfirm')}</Text>
              </TouchableOpacity>
              <TouchableOpacity
                style={[styles.actionButton, styles.cancelButton]}
                onPress={() => setShowDecline(false)}
                disabled={busy}
              >
                <Text style={[styles.actionButtonText, { color: colors.text }]}>{t('common.cancel')}</Text>
              </TouchableOpacity>
            </View>
          </View>
        </View>
      </Modal>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  scrollContent: {
    padding: theme.spacing.md,
  },
  offerCard: {
    backgroundColor: '#fff7ed',
    borderWidth: 1,
    borderColor: colors.accent,
    borderRadius: 12,
    padding: theme.spacing.md,
    marginBottom: theme.spacing.lg,
  },
  offerTitle: {
    fontSize: theme.fontSize.lg,
    fontWeight: 'bold',
    color: colors.text,
  },
  offerSub: {
    fontSize: theme.fontSize.sm,
    color: colors.textSecondary,
    marginTop: theme.spacing.xs,
    marginBottom: theme.spacing.md,
  },
  actionRow: {
    flexDirection: 'row',
    gap: theme.spacing.sm,
  },
  actionButton: {
    flex: 1,
    minHeight: 48,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    paddingHorizontal: theme.spacing.md,
  },
  actionButtonText: {
    color: colors.textWhite,
    fontSize: theme.fontSize.md,
    fontWeight: '700',
  },
  acceptButton: { backgroundColor: colors.success },
  declineButton: { backgroundColor: colors.backgroundWhite, borderWidth: 1, borderColor: colors.error },
  declineConfirmButton: { backgroundColor: colors.error },
  cancelButton: { backgroundColor: colors.backgroundWhite, borderWidth: 1, borderColor: colors.border },
  primaryAction: { backgroundColor: colors.accent, flex: 0, marginBottom: theme.spacing.lg },
  disabled: { opacity: 0.6 },
  modalBackdrop: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.4)',
    justifyContent: 'center',
    padding: theme.spacing.lg,
  },
  modalCard: {
    backgroundColor: colors.backgroundWhite,
    borderRadius: 12,
    padding: theme.spacing.lg,
  },
  modalLabel: {
    fontSize: theme.fontSize.sm,
    fontWeight: '600',
    color: colors.text,
    marginTop: theme.spacing.md,
    marginBottom: theme.spacing.xs,
  },
  modalInput: {
    backgroundColor: colors.inputBg,
    borderRadius: 6,
    padding: theme.spacing.md,
    minHeight: 80,
    textAlignVertical: 'top',
    fontSize: theme.fontSize.md,
    color: colors.text,
    marginBottom: theme.spacing.md,
  },
  centerContent: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: theme.spacing.lg,
  },
  loadingText: {
    fontSize: theme.fontSize.md,
    color: colors.textSecondary,
    marginTop: theme.spacing.md,
  },
  errorText: {
    fontSize: theme.fontSize.md,
    color: colors.error,
    textAlign: 'center',
    marginBottom: theme.spacing.md,
  },
  retryButton: {
    backgroundColor: colors.primary,
    paddingHorizontal: theme.spacing.lg,
    paddingVertical: theme.spacing.md,
    borderRadius: 8,
  },
  retryButtonText: {
    color: colors.textWhite,
    fontSize: theme.fontSize.md,
    fontWeight: '600',
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: theme.spacing.lg,
  },
  jobTitle: {
    fontSize: theme.fontSize.xl,
    fontWeight: 'bold',
    color: colors.text,
    flex: 1,
    marginRight: theme.spacing.sm,
  },
  statusBadge: {
    paddingHorizontal: theme.spacing.md,
    paddingVertical: theme.spacing.xs,
    borderRadius: 16,
  },
  statusText: {
    color: colors.textWhite,
    fontSize: theme.fontSize.sm,
    fontWeight: '600',
    textTransform: 'capitalize',
  },
  section: {
    marginBottom: theme.spacing.lg,
  },
  sectionTitle: {
    fontSize: theme.fontSize.lg,
    fontWeight: 'bold',
    color: colors.text,
    marginBottom: theme.spacing.sm,
  },
  descriptionText: {
    fontSize: theme.fontSize.md,
    color: colors.text,
    lineHeight: 22,
  },
  infoText: {
    fontSize: theme.fontSize.md,
    color: colors.text,
  },
  customerCard: {
    backgroundColor: colors.cardBackground,
    borderRadius: 12,
    padding: theme.spacing.md,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  customerName: {
    fontSize: theme.fontSize.lg,
    fontWeight: 'bold',
    color: colors.text,
    marginBottom: theme.spacing.sm,
  },
  infoRow: {
    flexDirection: 'row',
    marginBottom: theme.spacing.xs,
  },
  infoLabel: {
    fontSize: theme.fontSize.md,
    color: colors.textSecondary,
    minWidth: 70,
  },
  infoValue: {
    fontSize: theme.fontSize.md,
    color: colors.text,
    flex: 1,
  },
  linkText: {
    color: colors.primary,
    textDecorationLine: 'underline',
  },
  navigateButton: {
    backgroundColor: colors.primary,
    paddingVertical: theme.spacing.md,
    paddingHorizontal: theme.spacing.lg,
    borderRadius: 8,
    marginTop: theme.spacing.md,
    alignItems: 'center',
  },
  navigateButtonText: {
    color: colors.textWhite,
    fontSize: theme.fontSize.md,
    fontWeight: '600',
  },
  notesText: {
    fontSize: theme.fontSize.md,
    color: colors.textSecondary,
    fontStyle: 'italic',
    lineHeight: 22,
  },
});

export default ProjectDetailScreen;
