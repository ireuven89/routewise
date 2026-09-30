import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, ActivityIndicator, RefreshControl } from 'react-native';
import { useNavigation, useFocusEffect } from '@react-navigation/native';
import { jobs, storage } from '../../services/api';
import { colors, theme } from '../../theme/colors';
import { useLanguage, useRTL } from '../../i18n/LanguageContext';
import LanguageToggle from '../../components/LanguageToggle';

const ProjectsListScreen = () => {
  const navigation = useNavigation();
  const { t, locale } = useLanguage();
  const rtl = useRTL();
  const statusLabel = (status) => (status ? t(`status.${status.toLowerCase()}`) : t('status.pending'));
  const [jobsList, setJobsList] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState(null);

  // Reload whenever the list comes back into view (e.g. after accepting/declining a job).
  useFocusEffect(useCallback(() => {
    loadJobs();
  }, []));

  const loadJobs = async () => {
    try {
      setError(null);
      const response = await jobs.getMyJobs();
      // Offers waiting for an answer first.
      const list = [...(response || [])];
      list.sort((a, b) => (b.assignment_status === 'pending') - (a.assignment_status === 'pending'));
      setJobsList(list);
    } catch (err) {
      console.error('Failed to load jobs:', err);
      setError(t('projects.loadFailed'));
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleRefresh = () => {
    setRefreshing(true);
    loadJobs();
  };

  const handleLogout = async () => {
    await storage.clearAll();
    // Navigation will be handled by auth context
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
    return date.toLocaleDateString(locale, { month: 'short', day: 'numeric', year: 'numeric' });
  };

  const renderJobCard = ({ item }) => (
    <TouchableOpacity
      style={styles.jobCard}
      onPress={() => navigation.navigate('ProjectDetail', { jobId: item.id })}
    >
      {item.assignment_status === 'pending' && (
        <View style={[styles.newBadge, rtl.isRTL && { alignSelf: 'flex-end' }]} testID={`new-${item.id}`}>
          <Text style={styles.newBadgeText}>🔔 {t('projects.newRespond')}</Text>
        </View>
      )}
      <View style={[styles.jobHeader, rtl.row]}>
        <Text style={[styles.jobTitle, rtl.text]} numberOfLines={1}>
          {item.title || t('projects.jobNumber', { id: item.id })}
        </Text>
        <View style={[styles.statusBadge, { backgroundColor: getStatusColor(item.status) }]}>
          <Text style={styles.statusText}>{statusLabel(item.status)}</Text>
        </View>
      </View>

      <View style={[styles.jobInfo, rtl.row]}>
        <Text style={styles.jobLabel}>{t('projects.customer')}</Text>
        <Text style={[styles.jobValue, rtl.text]}>{item.customer?.name || t('projects.unknown')}</Text>
      </View>

      {!!item.customer?.address && (
        <View style={[styles.jobInfo, rtl.row]}>
          <Text style={styles.jobLabel}>📍</Text>
          <Text style={[styles.jobValue, rtl.text]} numberOfLines={2}>
            {item.customer.address}
          </Text>
        </View>
      )}

      <View style={[styles.jobInfo, rtl.row]}>
        <Text style={styles.jobLabel}>{t('projects.scheduled')}</Text>
        <Text style={[styles.jobValue, rtl.text]}>{formatDate(item.scheduled_at || item.scheduled_date)}</Text>
      </View>

      {!!item.description && (
        <Text style={[styles.jobDescription, rtl.text]} numberOfLines={2}>
          {item.description}
        </Text>
      )}
    </TouchableOpacity>
  );

  if (loading) {
    return (
      <View style={styles.container}>
        <View style={[styles.header, rtl.row]}>
          <Text style={styles.headerTitle}>{t('projects.myJobs')}</Text>
        </View>
        <View style={styles.centerContent}>
          <ActivityIndicator size="large" color={colors.primary} />
          <Text style={styles.loadingText}>{t('projects.loading')}</Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <View style={[styles.header, rtl.row]}>
        <Text style={styles.headerTitle}>{t('projects.myJobs')}</Text>
        <View style={[styles.headerActions, rtl.row]}>
          <LanguageToggle />
          <TouchableOpacity onPress={handleLogout}>
            <Text style={styles.logoutText}>{t('projects.logout')}</Text>
          </TouchableOpacity>
        </View>
      </View>

      {error ? (
        <View style={styles.centerContent}>
          <Text style={styles.errorText}>⚠️ {error}</Text>
          <TouchableOpacity style={styles.retryButton} onPress={loadJobs}>
            <Text style={styles.retryButtonText}>{t('common.retry')}</Text>
          </TouchableOpacity>
        </View>
      ) : jobsList.length === 0 ? (
        <View style={styles.centerContent}>
          <Text style={styles.emptyText}>📋 {t('projects.empty')}</Text>
          <Text style={styles.emptySubtext}>{t('projects.emptySub')}</Text>
        </View>
      ) : (
        <FlatList
          data={jobsList}
          renderItem={renderJobCard}
          keyExtractor={(item) => item.id.toString()}
          contentContainerStyle={styles.listContent}
          refreshControl={
            <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} colors={[colors.primary]} />
          }
        />
      )}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
  },
  header: {
    backgroundColor: colors.primary,
    paddingTop: 60,
    paddingBottom: 20,
    paddingHorizontal: theme.spacing.lg,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  headerTitle: {
    fontSize: theme.fontSize.xl,
    fontWeight: 'bold',
    color: colors.textWhite,
  },
  headerActions: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: theme.spacing.md,
  },
  logoutText: {
    color: colors.accent,
    fontSize: theme.fontSize.md,
    fontWeight: '600',
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
  emptyText: {
    fontSize: theme.fontSize.lg,
    color: colors.text,
    marginBottom: theme.spacing.sm,
  },
  emptySubtext: {
    fontSize: theme.fontSize.md,
    color: colors.textSecondary,
  },
  listContent: {
    padding: theme.spacing.md,
  },
  jobCard: {
    backgroundColor: colors.cardBackground,
    borderRadius: 12,
    padding: theme.spacing.md,
    marginBottom: theme.spacing.md,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  newBadge: {
    alignSelf: 'flex-start',
    backgroundColor: colors.accent,
    borderRadius: 12,
    paddingHorizontal: theme.spacing.sm,
    paddingVertical: 3,
    marginBottom: theme.spacing.sm,
  },
  newBadgeText: {
    color: colors.textWhite,
    fontSize: theme.fontSize.xs,
    fontWeight: '700',
  },
  jobHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: theme.spacing.sm,
  },
  jobTitle: {
    fontSize: theme.fontSize.lg,
    fontWeight: 'bold',
    color: colors.text,
    flex: 1,
    marginRight: theme.spacing.sm,
  },
  statusBadge: {
    paddingHorizontal: theme.spacing.sm,
    paddingVertical: 4,
    borderRadius: 12,
  },
  statusText: {
    color: colors.textWhite,
    fontSize: theme.fontSize.sm,
    fontWeight: '600',
    textTransform: 'capitalize',
  },
  jobInfo: {
    flexDirection: 'row',
    marginBottom: theme.spacing.xs,
  },
  jobLabel: {
    fontSize: theme.fontSize.md,
    color: colors.textSecondary,
    marginRight: theme.spacing.xs,
    minWidth: 80,
  },
  jobValue: {
    fontSize: theme.fontSize.md,
    color: colors.text,
    flex: 1,
  },
  jobDescription: {
    fontSize: theme.fontSize.sm,
    color: colors.textSecondary,
    marginTop: theme.spacing.sm,
    fontStyle: 'italic',
  },
});

export default ProjectsListScreen;
