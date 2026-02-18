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

const COUNTRY_CODES = [
  { code: '+1', flag: '🇺🇸', label: 'US/Canada' },
  { code: '+972', flag: '🇮🇱', label: 'Israel' },
  { code: '+44', flag: '🇬🇧', label: 'UK' },
  { code: '+61', flag: '🇦🇺', label: 'Australia' },
  { code: '+91', flag: '🇮🇳', label: 'India' },
  { code: '+49', flag: '🇩🇪', label: 'Germany' },
  { code: '+33', flag: '🇫🇷', label: 'France' },
  { code: '+52', flag: '🇲🇽', label: 'Mexico' },
];

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
  const [showCountryPicker, setShowCountryPicker] = useState(false);

  // Step 1: Request OTP
  const handleRequestOTP = async () => {
    if (!companyCode || !phoneNumber) {
      Alert.alert('Error', 'Please enter company code and phone number');
      return;
    }

    const fullPhone = formatPhone(countryCode, phoneNumber);

    setLoading(true);
    try {
      await auth.requestOTP(companyCode.trim().toUpperCase(), fullPhone);
      setStep(STEP_OTP);
    } catch (error) {
      Alert.alert('Error', error.response?.data?.error || 'Failed to send code');
    } finally {
      setLoading(false);
    }
  };

  // Step 2: Verify OTP
  const handleVerifyOTP = async () => {
    if (!otpCode || otpCode.length !== 6) {
      Alert.alert('Error', 'Please enter the 6-digit code');
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
      Alert.alert('Invalid Code', error.response?.data?.error || 'Code is invalid or expired');
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
    const fullPhone = countryCode + phoneNumber;
    setLoading(true);
    try {
      await auth.requestOTP(companyCode.trim().toUpperCase(), fullPhone);
      Alert.alert('Sent', 'Code sent again to your phone');
    } catch (error) {
      Alert.alert('Error', 'Failed to resend code');
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
          <Text style={styles.logo}>RouteWise</Text>
          <Text style={styles.subtitle}>Field Worker Login</Text>
        </View>

        <View style={styles.form}>
          <View style={styles.card}>

            {/* ========== STEP 1: Phone ========== */}
            {step === STEP_PHONE && (
                <>
                  <Text style={styles.welcomeText}>Welcome back</Text>
                  <Text style={styles.instructionText}>Enter your company code and phone number</Text>

                  {/* Company Code */}
                  <View style={styles.inputContainer}>
                    <Text style={styles.label}>Company Code</Text>
                    <TextInput
                        style={styles.input}
                        placeholder="e.g. ACME-HVAC-A7F3"
                        value={companyCode}
                        onChangeText={setCompanyCode}
                        autoCapitalize="characters"
                        autoCorrect={false}
                    />
                  </View>

                  {/* Phone Number with Country Code */}
                  <View style={styles.inputContainer}>
                    <Text style={styles.label}>Phone Number</Text>
                    <View style={styles.phoneRow}>
                      {/* Country Code Button */}
                      <TouchableOpacity
                          style={styles.countryCodeButton}
                          onPress={() => setShowCountryPicker(!showCountryPicker)}
                      >
                        <Text style={styles.countryCodeText}>
                          {COUNTRY_CODES.find((c) => c.code === countryCode)?.flag} {countryCode}
                        </Text>
                      </TouchableOpacity>

                      {/* Phone Input */}
                      <TextInput
                          style={styles.phoneInput}
                          placeholder="1234567890"
                          value={phoneNumber}
                          onChangeText={setPhoneNumber}
                          keyboardType="phone-pad"
                          autoCorrect={false}
                      />
                    </View>

                    {/* Country Picker Dropdown */}
                    {showCountryPicker && (
                        <View style={styles.countryPickerDropdown}>
                          {COUNTRY_CODES.map((item) => (
                              <TouchableOpacity
                                  key={item.code}
                                  style={[
                                    styles.countryPickerItem,
                                    countryCode === item.code && styles.countryPickerItemActive,
                                  ]}
                                  onPress={() => {
                                    setCountryCode(item.code);
                                    setShowCountryPicker(false);
                                  }}
                              >
                                <Text style={styles.countryPickerItemText}>
                                  {item.flag}  {item.code}  {item.label}
                                </Text>
                              </TouchableOpacity>
                          ))}
                        </View>
                    )}

                    {/* Preview full number */}
                    <Text style={styles.phonePreview}>
                      Full number: {countryCode}{phoneNumber}
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
                        <Text style={styles.primaryButtonText}>Send Code</Text>
                    )}
                  </TouchableOpacity>
                </>
            )}

            {/* ========== STEP 2: OTP ========== */}
            {step === STEP_OTP && (
                <>
                  <Text style={styles.welcomeText}>Enter your code</Text>
                  <Text style={styles.instructionText}>
                    We sent a 6-digit code to {countryCode}{phoneNumber}
                  </Text>

                  {/* OTP Input */}
                  <View style={styles.inputContainer}>
                    <Text style={styles.label}>Verification Code</Text>
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
                        <Text style={styles.primaryButtonText}>Verify</Text>
                    )}
                  </TouchableOpacity>

                  {/* Resend + Back */}
                  <View style={styles.otpActions}>
                    <TouchableOpacity onPress={handleResend} disabled={loading}>
                      <Text style={styles.resendText}>Resend Code</Text>
                    </TouchableOpacity>

                    <TouchableOpacity onPress={handleBack}>
                      <Text style={styles.backText}>← Back</Text>
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

  // Phone Row
  phoneRow: {
    flexDirection: 'row',
    gap: 8,
  },
  countryCodeButton: {
    backgroundColor: colors.inputBg,
    borderRadius: theme.borderRadius.sm,
    paddingHorizontal: theme.spacing.sm,
    minHeight: 48,
    justifyContent: 'center',
    alignItems: 'center',
    minWidth: 80,
  },
  countryCodeText: {
    fontSize: theme.fontSize.md,
    color: colors.text,
    fontWeight: '600',
  },
  phoneInput: {
    flex: 1,
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

  // Country Picker Dropdown
  countryPickerDropdown: {
    marginTop: 4,
    backgroundColor: colors.backgroundWhite,
    borderRadius: theme.borderRadius.sm,
    borderWidth: 1,
    borderColor: '#e0e0e0',
    maxHeight: 200,
    zIndex: 10,
  },
  countryPickerItem: {
    padding: theme.spacing.sm,
    borderBottomWidth: 1,
    borderBottomColor: '#f0f0f0',
  },
  countryPickerItemActive: {
    backgroundColor: '#e8f0fe',
  },
  countryPickerItemText: {
    fontSize: theme.fontSize.md,
    color: colors.text,
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