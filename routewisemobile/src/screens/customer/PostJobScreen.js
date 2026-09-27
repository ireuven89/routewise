import React, { useEffect, useState } from 'react';
import {
  View, Text, TextInput, TouchableOpacity, ScrollView, ActivityIndicator, KeyboardAvoidingView, Platform,
} from 'react-native';
import { publicApi, storage } from '../../services/api';
import { useLanguage, useRTL } from '../../i18n/LanguageContext';
import ScreenHeader from '../../components/ScreenHeader';
import PhoneInput from '../../components/PhoneInput';
import { shared } from './styles';
import { formatPhone } from '../../utils/phone';
import { notify } from '../../utils/confirm';
import { colors } from '../../theme/colors';

// Name + phone, then POST /public/service-requests. The returned access_token is the
// customer's only handle on the request, so it's saved on the device for "My requests".
const PostJobScreen = ({ navigation, route }) => {
  const { job } = route.params;
  const { t } = useLanguage();
  const rtl = useRTL();
  const [name, setName] = useState('');
  const [countryCode, setCountryCode] = useState('+972');
  const [phoneNumber, setPhoneNumber] = useState('');
  const [sending, setSending] = useState(false);

  useEffect(() => {
    storage.getCustomerProfile().then((p) => {
      if (!p) return;
      setName(p.name || '');
      setCountryCode(p.countryCode || '+972');
      setPhoneNumber(p.phoneNumber || '');
    });
  }, []);

  const handleSubmit = async () => {
    if (!name.trim() || !phoneNumber.trim()) {
      notify(t('common.error'), t('findService.nameAndPhoneRequired'));
      return;
    }
    setSending(true);
    try {
      const payload = {
        ...job,
        customer_name: name.trim(),
        customer_phone: formatPhone(countryCode, phoneNumber.trim()),
      };
      if (!payload.preferred_time) delete payload.preferred_time;

      const res = await publicApi.createRequest(payload);
      await storage.addMyRequest({
        token: res.access_token,
        id: res.id,
        service_type: job.service_type,
        address: job.address,
        created_at: new Date().toISOString(),
      });
      await storage.saveCustomerProfile({ name: name.trim(), countryCode, phoneNumber: phoneNumber.trim() });

      navigation.reset({
        index: 1,
        routes: [{ name: 'FindService' }, { name: 'RequestTracking', params: { token: res.access_token } }],
      });
    } catch {
      notify(t('common.error'), t('findService.requestError'));
    } finally {
      setSending(false);
    }
  };

  return (
    <KeyboardAvoidingView style={shared.container} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <ScreenHeader
        title={t('findService.postJob')}
        subtitle={t('findService.postJobSub')}
        onBack={() => navigation.goBack()}
      />
      <ScrollView contentContainerStyle={shared.scroll} keyboardShouldPersistTaps="handled">
        <View style={shared.card}>
          <Text style={[shared.hint, rtl.text, { marginTop: 0, marginBottom: 12 }]}>
            {t(`findService.${job.service_type}`)} · 📍 {job.address}
          </Text>

          <View style={shared.field}>
            <Text style={[shared.label, rtl.text]}>{t('findService.yourName')}</Text>
            <TextInput
              style={[shared.input, rtl.text]}
              placeholder={t('findService.namePlaceholder')}
              value={name}
              onChangeText={setName}
              testID="name-input"
            />
          </View>

          <View style={shared.field}>
            <Text style={[shared.label, rtl.text]}>{t('findService.yourPhone')}</Text>
            <PhoneInput
              countryCode={countryCode}
              onCountryCodeChange={setCountryCode}
              phoneNumber={phoneNumber}
              onPhoneNumberChange={setPhoneNumber}
              testID="phone-input"
            />
          </View>
        </View>

        <TouchableOpacity
          style={[shared.primaryButton, sending && shared.disabled]}
          onPress={handleSubmit}
          disabled={sending}
          testID="submit-request"
        >
          {sending
            ? <ActivityIndicator color={colors.textWhite} />
            : <Text style={shared.primaryButtonText}>{t('findService.postJobBtn')}</Text>}
        </TouchableOpacity>
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

export default PostJobScreen;
