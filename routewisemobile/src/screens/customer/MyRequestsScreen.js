import React, { useCallback, useState } from 'react';
import { View, Text, TouchableOpacity, FlatList, ActivityIndicator, RefreshControl, StyleSheet } from 'react-native';
import { useFocusEffect } from '@react-navigation/native';
import { publicApi, storage } from '../../services/api';
import { useLanguage, useRTL } from '../../i18n/LanguageContext';
import ScreenHeader from '../../components/ScreenHeader';
import { shared, requestStatusColor, requestStatusKey } from './styles';
import { formatDateTime } from '../../utils/format';
import { colors, theme } from '../../theme/colors';

// Requests posted from this device (tokens in storage), each refreshed from the API for current status + offer count.
const MyRequestsScreen = ({ navigation }) => {
  const { t, locale } = useLanguage();
  const rtl = useRTL();
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    const saved = await storage.getMyRequests();
    const results = await Promise.allSettled(saved.map((r) => publicApi.getRequest(r.token)));
    setItems(saved.map((r, i) => ({
      ...r,
      live: results[i].status === 'fulfilled' ? results[i].value : null,
    })));
    setLoading(false);
    setRefreshing(false);
  }, []);

  useFocusEffect(useCallback(() => { load(); }, [load]));

  const renderItem = ({ item }) => {
    const status = item.live?.request?.status;
    const offers = item.live?.bids?.length ?? 0;
    return (
      <TouchableOpacity
        style={shared.card}
        onPress={() => navigation.navigate('RequestTracking', { token: item.token })}
        testID={`my-request-${item.id}`}
      >
        <View style={[styles.rowBetween, rtl.row]}>
          <Text style={[styles.title, rtl.text]}>{t(`findService.${item.service_type}`)}</Text>
          <View style={[shared.badge, { backgroundColor: status ? requestStatusColor(status) : colors.disabled }]}>
            <Text style={shared.badgeText}>{status ? t(requestStatusKey(status)) : t('myRequests.unavailable')}</Text>
          </View>
        </View>
        {item.address ? <Text style={[shared.hint, rtl.text]}>📍 {item.address}</Text> : null}
        <View style={[styles.rowBetween, rtl.row, { marginTop: 6 }]}>
          <Text style={shared.hint}>{formatDateTime(item.created_at, locale)}</Text>
          {status ? <Text style={styles.offers}>{t('requestTracking.bidsHeading')}: {offers}</Text> : null}
        </View>
      </TouchableOpacity>
    );
  };

  return (
    <View style={shared.container}>
      <ScreenHeader title={t('myRequests.title')} onBack={() => navigation.goBack()} />
      {loading ? (
        <View style={shared.center}><ActivityIndicator size="large" color={colors.primary} /></View>
      ) : (
        <FlatList
          data={items}
          keyExtractor={(r) => r.token}
          renderItem={renderItem}
          contentContainerStyle={shared.scroll}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => { setRefreshing(true); load(); }} />}
          ListEmptyComponent={(
            <View style={{ alignItems: 'center', marginTop: 40 }}>
              <Text style={shared.muted}>{t('myRequests.empty')}</Text>
              <TouchableOpacity style={[shared.primaryButton, { marginTop: 16, paddingHorizontal: 24 }]} onPress={() => navigation.navigate('FindService')}>
                <Text style={shared.primaryButtonText}>{t('myRequests.newRequest')}</Text>
              </TouchableOpacity>
            </View>
          )}
        />
      )}
    </View>
  );
};

const styles = StyleSheet.create({
  rowBetween: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', gap: 8 },
  title: { fontSize: theme.fontSize.lg, fontWeight: 'bold', color: colors.text, flex: 1 },
  offers: { fontSize: theme.fontSize.sm, color: colors.primary, fontWeight: '600' },
});

export default MyRequestsScreen;
