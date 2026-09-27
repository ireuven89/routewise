import React, { useState } from 'react';
import {
  View, Text, TextInput, TouchableOpacity, ScrollView, ActivityIndicator, KeyboardAvoidingView, Platform,
} from 'react-native';
import { useLanguage, useRTL } from '../../i18n/LanguageContext';
import { useMode } from '../../navigation/ModeContext';
import ScreenHeader from '../../components/ScreenHeader';
import { shared, SERVICE_TYPES } from './styles';
import { PREFERRED_TIMES, preferredTimeToISO } from '../../utils/format';
import {
  getCurrentLocation, geocodeAddress, LocationPermissionError, AddressNotFoundError,
} from '../../utils/location';
import { notify } from '../../utils/confirm';
import { colors } from '../../theme/colors';

// Customer home: what, where, when. Then either post the job to nearby providers
// (primary, like the web) or preview the providers first.
const FindServiceScreen = ({ navigation }) => {
  const { t } = useLanguage();
  const rtl = useRTL();
  const { setMode } = useMode();

  const [serviceType, setServiceType] = useState('hvac');
  const [address, setAddress] = useState('');
  const [coords, setCoords] = useState(null); // set only by "use my location"
  const [description, setDescription] = useState('');
  const [preferredTime, setPreferredTime] = useState('timeFlexible');
  const [locating, setLocating] = useState(false);
  const [resolving, setResolving] = useState(false);

  const handleUseLocation = async () => {
    setLocating(true);
    try {
      const loc = await getCurrentLocation();
      setCoords({ latitude: loc.latitude, longitude: loc.longitude });
      setAddress(loc.address);
    } catch (err) {
      notify(t('common.error'), err instanceof LocationPermissionError
        ? t('findService.locationDenied')
        : t('findService.addressNotFound'));
    } finally {
      setLocating(false);
    }
  };

  const resolveLocation = async () => {
    if (coords) return { ...coords, address };
    if (!address.trim()) {
      notify(t('common.error'), t('findService.addressRequired'));
      return null;
    }
    try {
      return await geocodeAddress(address.trim());
    } catch (err) {
      notify(t('common.error'), err instanceof AddressNotFoundError
        ? t('findService.addressNotFound')
        : t('findService.requestError'));
      return null;
    }
  };

  const go = async (screen) => {
    setResolving(true);
    const location = await resolveLocation();
    setResolving(false);
    if (!location) return;
    navigation.navigate(screen, {
      job: {
        service_type: serviceType,
        description: description.trim(),
        latitude: location.latitude,
        longitude: location.longitude,
        address: location.address,
        preferred_time: preferredTimeToISO(preferredTime),
      },
    });
  };

  return (
    <KeyboardAvoidingView style={shared.container} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <ScreenHeader
        title={t('findService.title')}
        subtitle={t('findService.subtitle')}
        action={(
          <TouchableOpacity onPress={() => navigation.navigate('MyRequests')} testID="open-my-requests">
            <Text style={{ color: colors.accent, fontWeight: '700' }}>{t('findService.myRequests')}</Text>
          </TouchableOpacity>
        )}
      />

      <ScrollView contentContainerStyle={shared.scroll} keyboardShouldPersistTaps="handled">
        <View style={shared.card}>
          <Text style={[shared.label, rtl.text]}>{t('findService.serviceTypeLabel')}</Text>
          <View style={[shared.chipRow, rtl.row]}>
            {SERVICE_TYPES.map((s) => {
              const active = s.value === serviceType;
              return (
                <TouchableOpacity
                  key={s.value}
                  style={[shared.chip, active && shared.chipActive]}
                  onPress={() => setServiceType(s.value)}
                  testID={`service-${s.value}`}
                >
                  <Text style={[shared.chipText, active && shared.chipTextActive]}>
                    {s.icon} {t(`findService.${s.value}`)}
                  </Text>
                </TouchableOpacity>
              );
            })}
          </View>
        </View>

        <View style={shared.card}>
          <View style={shared.field}>
            <Text style={[shared.label, rtl.text]}>{t('findService.yourAddress')}</Text>
            <TextInput
              style={[shared.input, rtl.text]}
              placeholder={t('findService.addressPlaceholder')}
              value={address}
              onChangeText={(v) => { setAddress(v); setCoords(null); }}
              testID="address-input"
            />
            <TouchableOpacity
              style={shared.secondaryButton}
              onPress={handleUseLocation}
              disabled={locating}
              testID="use-location"
            >
              {locating
                ? <ActivityIndicator color={colors.primary} />
                : <Text style={shared.secondaryButtonText}>📍 {t('findService.useMyLocation')}</Text>}
            </TouchableOpacity>
            {coords ? <Text style={[shared.hint, rtl.text]}>✓ {t('findService.locationSet')}</Text> : null}
          </View>

          <View style={shared.field}>
            <Text style={[shared.label, rtl.text]}>{t('findService.problemLabel')}</Text>
            <TextInput
              style={[shared.input, rtl.text, { minHeight: 90, textAlignVertical: 'top' }]}
              placeholder={t('findService.problemPlaceholder')}
              value={description}
              onChangeText={setDescription}
              multiline
              testID="description-input"
            />
          </View>

          <Text style={[shared.label, rtl.text]}>{t('findService.preferredTime')}</Text>
          <View style={[shared.chipRow, rtl.row]}>
            {PREFERRED_TIMES.map((opt) => {
              const active = opt === preferredTime;
              return (
                <TouchableOpacity
                  key={opt}
                  style={[shared.chip, active && shared.chipActive]}
                  onPress={() => setPreferredTime(opt)}
                  testID={`time-${opt}`}
                >
                  <Text style={[shared.chipText, active && shared.chipTextActive]}>{t(`findService.${opt}`)}</Text>
                </TouchableOpacity>
              );
            })}
          </View>
        </View>

        <TouchableOpacity
          style={[shared.primaryButton, resolving && shared.disabled]}
          onPress={() => go('PostJob')}
          disabled={resolving}
          testID="post-job"
        >
          {resolving
            ? <ActivityIndicator color={colors.textWhite} />
            : <Text style={shared.primaryButtonText}>{t('findService.postJobBtn')}</Text>}
        </TouchableOpacity>
        <TouchableOpacity
          style={shared.secondaryButton}
          onPress={() => go('ProviderResults')}
          disabled={resolving}
          testID="search-providers"
        >
          <Text style={shared.secondaryButtonText}>🔍 {t('findService.searchBtn')}</Text>
        </TouchableOpacity>

        <TouchableOpacity onPress={() => setMode(null)} style={{ marginTop: 24, alignItems: 'center' }}>
          <Text style={shared.muted}>{t('common.switchRole')}</Text>
        </TouchableOpacity>
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

export default FindServiceScreen;
