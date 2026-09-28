import React, { useState } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  KeyboardAvoidingView,
  Platform,
  Alert,
  ActivityIndicator,
} from 'react-native';
import { auth, storage } from '../../services/api';
import { colors, theme } from '../../theme/colors';
import {formatPhone} from "../../utils/phone";
import { useLanguage, useRTL } from '../../i18n/LanguageContext';
import { useMode } from '../../navigation/ModeContext';
import LanguageToggle from '../../components/LanguageToggle';
import PhoneInput from '../../components/PhoneInput';

// Step 1: Enter company code + phone
// Step 2: Enter OTP code
const STEP_PHONE = 'phone';
const STEP_OTP = 'otp';

const WorkerLoginScreen = ({ navigation }) => {
  const [step, setStep] = useState(STEP_PHONE);
  const [companyCode, setCompanyCode] = useState('');
  const [countryCode, setCountryCode] = useState('+1');
  const [phoneNumber, setPhoneNumber] = useState('');
  const [otpCode, setOtpCode] = useState('');
  const [loading, setLoading] = useState(false);
  const { t } = useLanguage();
  const rtl = useRTL();
  const { setMode } = useMode();

  // Step 1: Request OTP
  const handleRequestOTP = async () => {
    if (!companyCode || !phoneNumber) {
      Alert.alert(t('common.error'), t('workerLogin.missingFields'));
      return;
    }

    const fullPhone = formatPhone(countryCode, phoneNumber);

    setLoading(true);
    try {
      await auth.requestOTP(companyCode.trim().toUpperCase(), fullPhone);
      setStep(STEP_OTP);
    } catch (error) {
      Alert.alert(t('common.error'), error.response?.data?.error || t('workerLogin.sendFailed'));
    } finally {
      setLoading(false);
    }
  };

  // Step 2: Verify OTP
  const handleVerifyOTP = async () => {
    if (!otpCode || otpCode.length !== 6) {
      Alert.alert(t('common.error'), t('workerLogin.invalidLength'));
      return;
    }

    const fullPhone = formatPhone(countryCode, phoneNumber);

    setLoading(true);
    try {
      const data = await auth.verifyOTP(companyCode.trim().toUpperCase(), fullPhone, otpCode);
      await storage.saveToken(data.token);
      await storage.saveWorker(data.worker);
      // Navigation handled by AppNavigator
    } catch (error) {
      Alert.alert(t('workerLogin.invalidCodeTitle'), error.response?.data?.error || t('workerLogin.invalidCode'));
    } finally {
      setLoading(false);
    }
  };

  // Go back to phone step
  const handleBack = () => {
    setStep(STEP_PHONE);
    setOtpCode('');
  };

  // Resend OTP
  const handleResend = async () => {
    const fullPhone = formatPhone(countryCode, phoneNumber);
    setLoading(true);
    try {
      await auth.requestOTP(companyCode.trim().toUpperCase(), fullPhone);
      Alert.alert(t('workerLogin.resentTitle'), t('workerLogin.resent'));
    } catch (error) {
      Alert.alert(t('common.error'), t('workerLogin.resendFailed'));
    } finally {
      setLoading(false);
    }
  };

  return (
      <KeyboardAvoidingView
          style={styles.container}
          behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
      >
        {/* Header */}
        <View style={styles.header}>
          <View style={[styles.headerTop, rtl.row]}>
            <TouchableOpacity onPress={() => setMode(null)} hitSlop={12}>
              <Text style={styles.switchRole}>{rtl.arrowBack} {t('common.switchRole')}</Text>
            </TouchableOpacity>
            <LanguageToggle />
          </View>
          <Text style={[styles.logo, rtl.text]}>RouteWise</Text>
          <Text style={[styles.subtitle, rtl.text]}>{t('workerLogin.subtitle')}</Text>
        </View>

        <View style={styles.form}>
          <View style={styles.card}>

            {/* ========== STEP 1: Phone ========== */}
            {step === STEP_PHONE && (
                <>
                  <Text style={[styles.welcomeText, rtl.text]}>{t('workerLogin.welcome')}</Text>
                  <Text style={[styles.instructionText, rtl.text]}>{t('workerLogin.instruction')}</Text>

                  {/* Company Code */}
                  <View style={styles.inputContainer}>
                    <Text style={[styles.label, rtl.text]}>{t('workerLogin.companyCode')}</Text>
                    <TextInput
                        style={styles.input}
                        placeholder={t('workerLogin.companyCodePlaceholder')}
                        value={companyCode}
                        onChangeText={setCompanyCode}
                        autoCapitalize="characters"
                        autoCorrect={false}
                    />
                  </View>

                  {/* Phone Number with Country Code */}
                  <View style={styles.inputContainer}>
                    <Text style={[styles.label, rtl.text]}>{t('workerLogin.phone')}</Text>
                    <PhoneInput
                        countryCode={countryCode}
                        onCountryCodeChange={setCountryCode}
                        phoneNumber={phoneNumber}
                        onPhoneNumberChange={setPhoneNumber}
                    />

                    {/* Preview full number */}
                    <Text style={[styles.phonePreview, rtl.text]}>
                      {t('workerLogin.fullNumber', { phone: formatPhone(countryCode, phoneNumber) })}
                    </Text>
                  </View>

                  {/* Send Code Button */}
                  <TouchableOpacity
                      style={[styles.primaryButton, loading && styles.primaryButtonDisabled]}
                      onPress={handleRequestOTP}
                      disabled={loading}
                  >
                    {loading ? (
                        <ActivityIndicator color={colors.textWhite} />
                    ) : (
                        <Text style={styles.primaryButtonText}>{t('workerLogin.sendCode')}</Text>
                    )}
                  </TouchableOpacity>
                </>
            )}

            {/* ========== STEP 2: OTP ========== */}
            {step === STEP_OTP && (
                <>
                  <Text style={[styles.welcomeText, rtl.text]}>{t('workerLogin.enterCode')}</Text>
                  <Text style={[styles.instructionText, rtl.text]}>
                    {t('workerLogin.codeSentTo', { phone: formatPhone(countryCode, phoneNumber) })}
                  </Text>

                  {/* OTP Input */}
                  <View style={styles.inputContainer}>
                    <Text style={[styles.label, rtl.text]}>{t('workerLogin.verificationCode')}</Text>
                    <TextInput
                        style={[styles.input, styles.otpInput]}
                        placeholder="• • • • • •"
                        value={otpCode}
                        onChangeText={(text) => setOtpCode(text.replace(/\D/g, '').slice(0, 6))}
                        keyboardType="number-pad"
                        maxLength={6}
                        autoFocus
                    />
                  </View>

                  {/* Verify Button */}
                  <TouchableOpacity
                      style={[styles.primaryButton, loading && styles.primaryButtonDisabled]}
                      onPress={handleVerifyOTP}
                      disabled={loading}
                  >
                    {loading ? (
                        <ActivityIndicator color={colors.textWhite} />
                    ) : (
                        <Text style={styles.primaryButtonText}>{t('workerLogin.verify')}</Text>
                    )}
                  </TouchableOpacity>

                  {/* Resend + Back */}
                  <View style={styles.otpActions}>
                    <TouchableOpacity onPress={handleResend} disabled={loading}>
                      <Text style={styles.resendText}>{t('workerLogin.resend')}</Text>
                    </TouchableOpacity>

                    <TouchableOpacity onPress={handleBack}>
                      <Text style={styles.backText}>{rtl.arrowBack} {t('common.back')}</Text>
                    </TouchableOpacity>
                  </View>
                </>
            )}
          </View>
        </View>
      </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.primary },

  // Header
  header: {
    paddingTop: 60,
    paddingHorizontal: theme.spacing.lg,
    paddingBottom: theme.spacing.xl,
  },
  headerTop: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: theme.spacing.md,
  },
  switchRole: {
    color: colors.textWhite,
    opacity: 0.85,
    fontSize: theme.fontSize.sm,
  },
  logo: {
    fontSize: theme.fontSize.xxl,
    fontWeight: 'bold',
    color: colors.textWhite,
    marginBottom: theme.spacing.sm,
  },
  subtitle: {
    fontSize: theme.fontSize.lg,
    color: colors.textWhite,
    opacity: 0.9,
  },

  // Form
  form: {
    flex: 1,
    backgroundColor: colors.background,
    borderTopLeftRadius: 24,
    borderTopRightRadius: 24,
    paddingTop: theme.spacing.xl,
    paddingHorizontal: theme.spacing.lg,
  },
  card: {
    backgroundColor: colors.backgroundWhite,
    borderRadius: theme.borderRadius.lg,
    padding: theme.spacing.lg,
    ...theme.shadow.md,
  },
  welcomeText: {
    fontSize: theme.fontSize.xl,
    fontWeight: 'bold',
    color: colors.text,
    marginBottom: theme.spacing.xs,
  },
  instructionText: {
    fontSize: theme.fontSize.md,
    color: colors.textSecondary,
    marginBottom: theme.spacing.lg,
  },

  // Input
  inputContainer: { marginBottom: theme.spacing.md },
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


  phonePreview: {
    fontSize: theme.fontSize.sm,
    color: colors.textSecondary,
    marginTop: 6,
  },


  // OTP Input
  otpInput: {
    textAlign: 'center',
    fontSize: theme.fontSize.xl,
    fontWeight: 'bold',
    letterSpacing: 8,
  },

  // Buttons
  primaryButton: {
    backgroundColor: colors.accent,
    borderRadius: theme.borderRadius.sm,
    padding: theme.spacing.md,
    alignItems: 'center',
    marginTop: theme.spacing.md,
    minHeight: 48,
    justifyContent: 'center',
  },
  primaryButtonDisabled: { opacity: 0.6 },
  primaryButtonText: {
    color: colors.textWhite,
    fontSize: theme.fontSize.md,
    fontWeight: '600',
  },

  // OTP Actions
  otpActions: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: theme.spacing.md,
  },
  resendText: {
    color: colors.accent,
    fontSize: theme.fontSize.sm,
    fontWeight: '600',
  },
  backText: {
    color: colors.textSecondary,
    fontSize: theme.fontSize.sm,
  },
});

export default WorkerLoginScreen;