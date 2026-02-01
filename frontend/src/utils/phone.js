

export const formatPhone = (countryCode, phoneNumber) => {
    const cleaned = phoneNumber.replace(/^0/, '');
    return countryCode + cleaned;
};