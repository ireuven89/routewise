import React, { useEffect, useState } from 'react';
import { View, Text, TouchableOpacity, FlatList, ActivityIndicator, Linking, StyleSheet } from 'react-native';
import { publicApi } from '../../services/api';
import { useLanguage, useRTL } from '../../i18n/LanguageContext';
import ScreenHeader from '../../components/ScreenHeader';
import { shared } from './styles';
import { formatMoney } from '../../utils/format';
import { colors, theme } from '../../theme/colors';

const ProviderResultsScreen = ({ navigation, route }) => {
  const { job } = route.params;
  const { t } = useLanguage();
  const rtl = useRTL();
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  const load = async () => {
    setLoading(true);
    setError(false);
    try {
      setProviders(await publicApi.searchProviders(job.latitude, job.longitude, job.service_type));
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const renderProvider = ({ item }) => (
    <View style={shared.card} testID={`provider-${item.id}`}>
      <View style={[styles.rowBetween, rtl.row]}>
        <Text style={[styles.name, rtl.text]}>{item.name}</Text>
        {item.distance_km != null ? (
          <Text style={styles.distance}>{Number(item.distance_km).toFixed(1)} {t('findService.away')}</Text>
        ) : null}
      </View>
      {item.address ? <Text style={[shared.hint, rtl.text]}>📍 {item.address}</Text> : null}

      <View style={[styles.prices, rtl.row]}>
        {item.visit_fee != null ? (
          <View style={styles.price}>
            <Text style={styles.priceLabel}>{t('findService.visitFee')}</Text>
            <Text style={styles.priceValue}>{formatMoney(item.visit_fee)}</Text>
          </View>
        ) : null}
        {item.repair_estimate_min != null ? (
          <View style={styles.price}>
            <Text style={styles.priceLabel}>{t('findService.repairEstimate')}</Text>
            <Text style={styles.priceValue}>
              {formatMoney(item.repair_estimate_min)}–{formatMoney(item.repair_estimate_max)}
            </Text>
          </View>
        ) : null}
      </View>

      {item.phone ? (
        <TouchableOpacity style={shared.secondaryButton} onPress={() => Linking.openURL(`tel:${item.phone}`)}>
          <Text style={shared.secondaryButtonText}>📞 {t('findService.call')}</Text>
        </TouchableOpacity>
      ) : null}
    </View>
  );

  return (
    <View style={shared.container}>
      <ScreenHeader
        title={t(`findService.${job.service_type}`)}
        subtitle={loading ? t('findService.searching') : `${providers.length} ${t('findService.resultsFound')}`}
        onBack={() => navigation.goBack()}
      />

      {loading ? (
        <View style={shared.center}><ActivityIndicator size="large" color={colors.primary} /></View>
      ) : error ? (
        <View style={shared.center}>
          <Text style={shared.errorText}>{t('findService.requestError')}</Text>
          <TouchableOpacity style={shared.secondaryButton} onPress={load}>
            <Text style={shared.secondaryButtonText}>{t('common.retry')}</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <FlatList
          data={providers}
          keyExtractor={(p) => String(p.id)}
          renderItem={renderProvider}
          contentContainerStyle={shared.scroll}
          ListEmptyComponent={<Text style={[shared.muted, { marginTop: 32 }]}>{t('findService.noResults')}</Text>}
        />
      )}

      <View style={styles.footer}>
        <TouchableOpacity
          style={shared.primaryButton}
          onPress={() => navigation.navigate('PostJob', { job })}
          testID="post-job-from-results"
        >
          <Text style={shared.primaryButtonText}>{t('findService.postJobBtn')}</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  rowBetween: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  name: { fontSize: theme.fontSize.lg, fontWeight: 'bold', color: colors.text, flex: 1 },
  distance: { fontSize: theme.fontSize.sm, color: colors.textSecondary },
  prices: { flexDirection: 'row', gap: theme.spacing.md, marginTop: theme.spacing.sm },
  price: { flex: 1, backgroundColor: colors.background, borderRadius: theme.borderRadius.md, padding: theme.spacing.sm },
  priceLabel: { fontSize: theme.fontSize.xs, color: colors.textSecondary },
  priceValue: { fontSize: theme.fontSize.md, fontWeight: 'bold', color: colors.text, marginTop: 2 },
  footer: {
    padding: theme.spacing.md,
    backgroundColor: colors.backgroundWhite,
    borderTopWidth: 1,
    borderTopColor: colors.border,
  },
});

export default ProviderResultsScreen;
