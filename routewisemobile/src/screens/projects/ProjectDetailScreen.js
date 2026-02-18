import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, ScrollView, TouchableOpacity, ActivityIndicator, Linking, Platform, Alert } from 'react-native';
import { jobs } from '../../services/api';
import { colors, theme } from '../../theme/colors';

const ProjectDetailScreen = ({ navigation, route }) => {
  const { jobId } = route.params;
  const [job, setJob] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

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
      setError('Failed to load job details');
    } finally {
      setLoading(false);
    }
  };

  const handleNavigateToCustomer = async () => {
    const { customer } = job;

    if (!customer) {
      Alert.alert('Error', 'Customer information not available');
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
        Alert.alert('Error', 'Failed to open navigation app');
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
        Alert.alert('Error', 'Failed to open navigation app');
      }
    } else {
      Alert.alert('Error', 'No location information available for this customer');
    }
  };

  const handleCallCustomer = async () => {
    const { customer } = job;
    if (!customer?.phone) {
      Alert.alert('Error', 'Customer phone number not available');
      return;
    }

    const url = `tel:${customer.phone}`;
    try {
      const supported = await Linking.canOpenURL(url);
      if (supported) {
        await Linking.openURL(url);
      } else {
        Alert.alert('Error', 'Unable to make phone calls');
      }
    } catch (err) {
      console.error('Failed to call customer:', err);
      Alert.alert('Error', 'Failed to initiate call');
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
    if (!dateString) return 'Not scheduled';
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
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
          <Text style={styles.loadingText}>Loading job details...</Text>
        </View>
      </View>
    );
  }

  if (error || !job) {
    return (
      <View style={styles.container}>
        <View style={styles.centerContent}>
          <Text style={styles.errorText}>⚠️ {error || 'Job not found'}</Text>
          <TouchableOpacity style={styles.retryButton} onPress={loadJobDetails}>
            <Text style={styles.retryButtonText}>Retry</Text>
          </TouchableOpacity>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <ScrollView contentContainerStyle={styles.scrollContent}>
        {/* Job Header */}
        <View style={styles.header}>
          <Text style={styles.jobTitle}>{job.title || `Job #${job.id}`}</Text>
          <View style={[styles.statusBadge, { backgroundColor: getStatusColor(job.status) }]}>
            <Text style={styles.statusText}>{job.status || 'Pending'}</Text>
          </View>
        </View>

        {/* Job Description */}
        {job.description && (
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Description</Text>
            <Text style={styles.descriptionText}>{job.description}</Text>
          </View>
        )}

        {/* Schedule */}
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Schedule</Text>
          <Text style={styles.infoText}>📅 {formatDate(job.scheduled_date)}</Text>
        </View>

        {/* Customer Information */}
        {job.customer && (
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Customer</Text>
            <View style={styles.customerCard}>
              <Text style={styles.customerName}>{job.customer.name}</Text>

              {job.customer.phone && (
                <TouchableOpacity style={styles.infoRow} onPress={handleCallCustomer}>
                  <Text style={styles.infoLabel}>📞 Phone:</Text>
                  <Text style={[styles.infoValue, styles.linkText]}>{job.customer.phone}</Text>
                </TouchableOpacity>
              )}

              {job.customer.email && (
                <View style={styles.infoRow}>
                  <Text style={styles.infoLabel}>✉️ Email:</Text>
                  <Text style={styles.infoValue}>{job.customer.email}</Text>
                </View>
              )}

              {job.customer.address && (
                <View style={styles.infoRow}>
                  <Text style={styles.infoLabel}>📍 Address:</Text>
                  <Text style={styles.infoValue}>{job.customer.address}</Text>
                </View>
              )}

              {/* Navigation Button */}
              {(job.customer.address || (job.customer.latitude && job.customer.longitude)) && (
                <TouchableOpacity style={styles.navigateButton} onPress={handleNavigateToCustomer}>
                  <Text style={styles.navigateButtonText}>🗺️ Navigate to Customer</Text>
                </TouchableOpacity>
              )}
            </View>
          </View>
        )}

        {/* Notes */}
        {job.notes && (
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Notes</Text>
            <Text style={styles.notesText}>{job.notes}</Text>
          </View>
        )}
      </ScrollView>
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
