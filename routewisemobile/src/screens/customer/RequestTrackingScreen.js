import React, { useCallback, useEffect, useState } from 'react';
import {
  View, Text, TouchableOpacity, ScrollView, ActivityIndicator, RefreshControl, StyleSheet,
} from 'react-native';
import { publicApi } from '../../services/api';
import { useLanguage, useRTL } from '../../i18n/LanguageContext';
import ScreenHeader from '../../components/ScreenHeader';
import { shared, requestStatusColor, requestStatusKey } from './styles';
import { formatMoney } from '../../utils/format';
import { confirm, notify } from '../../utils/confirm';
import { colors, theme } from '../../theme/colors';

export const POLL_INTERVAL_MS = 15000;

const RequestTrackingScreen = ({ navigation, route }) => {
  const { token } = route.params;
  const { t } = useLanguage();
  const rtl = useRTL();
  const [request, setRequest] = useState(null);
  const [bids, setBids] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [notFound, setNotFound] = useState(false);
  const [awardingId, setAwardingId] = useState(null);

  const load = useCallback(async () => {
    try {
      const data = await publicApi.getRequest(token);
      setRequest(data.request);
      setBids(data.bids || []);
      setNotFound(false);
    } catch {
      setNotFound(true);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [token]);

  useEffect(() => { load(); }, [load]);

  // New offers arrive while the customer waits: poll only while the request is still open.
  useEffect(() => {
    if (request?.status !== 'open') return undefined;
    const id = setInterval(load, POLL_INTERVAL_MS);
    return () => clearInterval(id);
  }, [request?.status, load]);

  const handleAward = async (bid) => {
    const ok = await confirm({
      title: t('requestTracking.awardConfirmTitle'),
      message: t('requestTracking.awardConfirmBody', { name: bid.organization_name, price: formatMoney(bid.price) }),
      confirmText: t('requestTracking.award'),
      cancelText: t('common.cancel'),
    });
    if (!ok) return;
    setAwardingId(bid.id);
    try {
      await publicApi.awardBid(token, bid.id);
      await load();
    } catch {
      notify(t('common.error'), t('requestTracking.awardError'));
    } finally {
      setAwardingId(null);
    }
  };

  const awardedBid = bids.find((b) => b.status === 'awarded');

  const renderBid = (bid) => {
    const canAward = request.status === 'open' && bid.status === 'submitted';
    return (
      <View key={bid.id} style={[shared.card, bid.status === 'awarded' && styles.awardedCard]} testID={`bid-${bid.id}`}>
        <View style={[styles.rowBetween, rtl.row]}>
          <Text style={[styles.orgName, rtl.text]}>{bid.organization_name}</Text>
          <Text style={styles.price}>{formatMoney(bid.price)}</Text>
        </View>
        {bid.eta_minutes != null ? (
          <Text style={[shared.hint, rtl.text]}>
            ⏱ {t('requestTracking.eta')}: {bid.eta_minutes} {t('requestTracking.minutes')}
          </Text>
        ) : null}
        {bid.message ? <Text style={[styles.message, rtl.text]}>“{bid.message}”</Text> : null}

        {bid.status === 'awarded' || bid.status === 'rejected' ? (
          <View style={[shared.badge, { backgroundColor: bid.status === 'awarded' ? colors.success : colors.textSecondary, marginTop: 8 }, rtl.isRTL && { alignSelf: 'flex-end' }]}>
            <Text style={shared.badgeText}>
              {t(bid.status === 'awarded' ? 'requestTracking.bidAwarded' : 'requestTracking.bidRejected')}
            </Text>
          </View>
        ) : null}

        {canAward ? (
          <TouchableOpacity
            style={[shared.primaryButton, { marginTop: 12 }, awardingId && shared.disabled]}
            onPress={() => handleAward(bid)}
            disabled={!!awardingId}
            testID={`award-${bid.id}`}
          >
            {awardingId === bid.id
              ? <ActivityIndicator color={colors.textWhite} />
              : <Text style={shared.primaryButtonText}>{t('requestTracking.award')}</Text>}
          </TouchableOpacity>
        ) : null}
      </View>
    );
  };

  let body;
  if (loading) {
    body = (
      <View style={shared.center}>
        <ActivityIndicator size="large" color={colors.primary} />
        <Text style={[shared.muted, { marginTop: 12 }]}>{t('requestTracking.loading')}</Text>
      </View>
    );
  } else if (notFound || !request) {
    body = (
      <View style={shared.center}>
        <Text style={shared.errorText}>{t('requestTracking.notFound')}</Text>
        <TouchableOpacity style={shared.secondaryButton} onPress={load}>
          <Text style={shared.secondaryButtonText}>{t('common.retry')}</Text>
        </TouchableOpacity>
      </View>
    );
  } else {
    body = (
      <ScrollView
        contentContainerStyle={shared.scroll}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => { setRefreshing(true); load(); }} />}
      >
        <View style={shared.card}>
          <View style={[styles.rowBetween, rtl.row]}>
            <Text style={[styles.serviceType, rtl.text]}>{t(`findService.${request.service_type}`)}</Text>
            <View style={[shared.badge, { backgroundColor: requestStatusColor(request.status) }]} testID="request-status">
              <Text style={shared.badgeText}>{t(requestStatusKey(request.status))}</Text>
            </View>
          </View>
          {request.address ? <Text style={[shared.hint, rtl.text]}>📍 {request.address}</Text> : null}
          {request.description ? <Text style={[styles.message, rtl.text]}>{request.description}</Text> : null}
        </View>

        {awardedBid ? (
          <View style={styles.awardedBanner}>
            <Text style={[styles.awardedText, rtl.text]}>
              ✓ {t('requestTracking.awardedTo', { name: awardedBid.organization_name })}
            </Text>
          </View>
        ) : null}

        <View style={[styles.rowBetween, rtl.row, { marginBottom: 8 }]}>
          <Text style={[styles.heading, rtl.text]}>{t('requestTracking.bidsHeading')} ({bids.length})</Text>
          <TouchableOpacity onPress={() => { setRefreshing(true); load(); }} testID="refresh">
            <Text style={styles.refresh}>↻ {t('requestTracking.refresh')}</Text>
          </TouchableOpacity>
        </View>

        {bids.length === 0
          ? <Text style={[shared.muted, { marginTop: 16 }]}>{t('requestTracking.noBidsYet')}</Text>
          : bids.map(renderBid)}
      </ScrollView>
    );
  }

  return (
    <View style={shared.container}>
      <ScreenHeader title={t('requestTracking.title')} onBack={() => navigation.goBack()} />
      {body}
    </View>
  );
};

const styles = StyleSheet.create({
  rowBetween: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', gap: 8 },
  serviceType: { fontSize: theme.fontSize.lg, fontWeight: 'bold', color: colors.text, flex: 1 },
  orgName: { fontSize: theme.fontSize.lg, fontWeight: 'bold', color: colors.text, flex: 1 },
  price: { fontSize: theme.fontSize.xl, fontWeight: 'bold', color: colors.primary },
  message: { fontSize: theme.fontSize.md, color: colors.text, marginTop: theme.spacing.sm },
  heading: { fontSize: theme.fontSize.lg, fontWeight: 'bold', color: colors.text },
  refresh: { color: colors.primary, fontWeight: '600' },
  awardedCard: { borderWidth: 2, borderColor: colors.success },
  awardedBanner: {
    backgroundColor: '#d1fae5',
    borderRadius: theme.borderRadius.md,
    padding: theme.spacing.md,
    marginBottom: theme.spacing.md,
  },
  awardedText: { color: '#065f46', fontWeight: '600', fontSize: theme.fontSize.md },
});

export default RequestTrackingScreen;
