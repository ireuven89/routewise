import React, { useState, useEffect } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, FlatList, ActivityIndicator, RefreshControl } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { jobs, storage } from '../../services/api';
import { colors, theme } from '../../theme/colors';

const ProjectsListScreen = () => {
  const navigation = useNavigation();
  const [jobsList, setJobsList] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    loadJobs();
  }, []);

  const loadJobs = async () => {
    try {
      setError(null);
      const response = await jobs.getMyJobs();
      setJobsList(response || []);
    } catch (err) {
      console.error('Failed to load jobs:', err);
      setError('Failed to load jobs. Pull to refresh.');
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
    if (!dateString) return 'Not scheduled';
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  };

  const renderJobCard = ({ item }) => (
    <TouchableOpacity
      style={styles.jobCard}
      onPress={() => navigation.navigate('ProjectDetail', { jobId: item.id })}
    >
      <View style={styles.jobHeader}>
        <Text style={styles.jobTitle} numberOfLines={1}>
          {item.title || `Job #${item.id}`}
        </Text>
        <View style={[styles.statusBadge, { backgroundColor: getStatusColor(item.status) }]}>
          <Text style={styles.statusText}>{item.status || 'Pending'}</Text>
        </View>
      </View>

      <View style={styles.jobInfo}>
        <Text style={styles.jobLabel}>Customer:</Text>
        <Text style={styles.jobValue}>{item.customer?.name || 'Unknown'}</Text>
      </View>

      {item.customer?.address && (
        <View style={styles.jobInfo}>
          <Text style={styles.jobLabel}>📍</Text>
          <Text style={styles.jobValue} numberOfLines={2}>
            {item.customer.address}
          </Text>
        </View>
      )}

      <View style={styles.jobInfo}>
        <Text style={styles.jobLabel}>Scheduled:</Text>
        <Text style={styles.jobValue}>{formatDate(item.scheduled_date)}</Text>
      </View>

      {item.description && (
        <Text style={styles.jobDescription} numberOfLines={2}>
          {item.description}
        </Text>
      )}
    </TouchableOpacity>
  );

  if (loading) {
    return (
      <View style={styles.container}>
        <View style={styles.header}>
          <Text style={styles.headerTitle}>My Jobs</Text>
        </View>
        <View style={styles.centerContent}>
          <ActivityIndicator size="large" color={colors.primary} />
          <Text style={styles.loadingText}>Loading jobs...</Text>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>My Jobs</Text>
        <TouchableOpacity onPress={handleLogout}>
          <Text style={styles.logoutText}>Logout</Text>
        </TouchableOpacity>
      </View>

      {error ? (
        <View style={styles.centerContent}>
          <Text style={styles.errorText}>⚠️ {error}</Text>
          <TouchableOpacity style={styles.retryButton} onPress={loadJobs}>
            <Text style={styles.retryButtonText}>Retry</Text>
          </TouchableOpacity>
        </View>
      ) : jobsList.length === 0 ? (
        <View style={styles.centerContent}>
          <Text style={styles.emptyText}>📋 No jobs assigned yet</Text>
          <Text style={styles.emptySubtext}>Check back later for new assignments</Text>
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
