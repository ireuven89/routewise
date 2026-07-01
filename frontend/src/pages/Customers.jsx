import { useState, useEffect, useCallback } from 'react';
import { customersAPI } from '../api/client';
import Layout from '../components/Layout';
import { useLanguage } from '../context/LanguageContext';
import CustomerModal from '../components/CustomerModal';
import { FaUsers, FaPhone, FaEnvelope, FaMapMarkerAlt, FaSearch } from 'react-icons/fa';

const getInitials = (name) => {
    if (!name) return '?';
    const parts = name.trim().split(/\s+/);
    if (parts.length === 1) return parts[0][0].toUpperCase();
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
};

const Customers = () => {
    const { t } = useLanguage();
    const [customers, setCustomers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [editingCustomer, setEditingCustomer] = useState(null);
    const [searchTerm, setSearchTerm] = useState('');

    const loadCustomers = useCallback(async () => {
        try {
            setLoading(true);
            const response = await customersAPI.getAll(searchTerm);
            setCustomers(response.data);
        } catch (error) {
            console.error('Failed to load customers:', error);
        } finally {
            setLoading(false);
        }
    }, [searchTerm]);

    useEffect(() => {
        loadCustomers();
    }, [loadCustomers]);

    const handleCreate = async (customerData) => {
        try {
            await customersAPI.create(customerData);
            await loadCustomers();
            setShowModal(false);
        } catch (error) {
            console.error('Failed to create customer:', error);
            alert(t('customers.failedCreate'));
        }
    };

    const handleUpdate = async (customerData) => {
        try {
            await customersAPI.update(editingCustomer.id, customerData);
            await loadCustomers();
            setEditingCustomer(null);
        } catch (error) {
            console.error('Failed to update customer:', error);
            alert(t('customers.failedUpdate'));
        }
    };

    const handleDelete = async (customerId) => {
        if (!window.confirm(t('customers.deleteConfirm'))) return;
        try {
            await customersAPI.delete(customerId);
            await loadCustomers();
        } catch (error) {
            console.error('Failed to delete customer:', error);
            alert(t('customers.failedDelete'));
        }
    };

    if (loading) {
        return (
            <Layout>
                <div className="px-4 sm:px-0">
                    <div className="flex justify-between items-center mb-6">
                        <div className="h-8 w-36 bg-gray-200 rounded-lg animate-pulse" />
                        <div className="h-10 w-36 bg-gray-200 rounded-xl animate-pulse" />
                    </div>
                    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
                        {[1, 2, 3].map(i => (
                            <div key={i} className="flex items-center gap-4 px-6 py-4 border-b border-gray-50 last:border-0">
                                <div className="w-10 h-10 rounded-full bg-gray-200 animate-pulse flex-shrink-0" />
                                <div className="flex-1 space-y-2">
                                    <div className="h-4 w-36 bg-gray-200 rounded animate-pulse" />
                                    <div className="h-3 w-52 bg-gray-100 rounded animate-pulse" />
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <div className="flex items-center gap-3">
                        <h1 className="text-2xl font-bold text-gray-900">{t('customers.title')}</h1>
                        <span className="text-sm font-medium text-gray-400 bg-gray-100 px-2.5 py-0.5 rounded-full">
                            {customers.length}
                        </span>
                    </div>
                    <button
                        onClick={() => setShowModal(true)}
                        className="bg-[#ff6b35] hover:opacity-90 text-white px-4 py-2.5 rounded-xl text-sm font-semibold shadow-sm transition-opacity"
                    >
                        {t('customers.addCustomer')}
                    </button>
                </div>

                {/* Search */}
                <div className="mb-6">
                    <div className="relative max-w-sm">
                        <FaSearch className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400 pointer-events-none" />
                        <input
                            type="text"
                            placeholder={t('customers.searchPlaceholder')}
                            value={searchTerm}
                            onChange={(e) => setSearchTerm(e.target.value)}
                            className="w-full pl-9 pr-4 py-2 text-sm border border-gray-200 rounded-xl bg-white shadow-sm focus:outline-none focus:ring-2 focus:ring-[#1e3a5f] focus:ring-opacity-20"
                        />
                    </div>
                </div>

                {/* Customers List */}
                {customers.length === 0 ? (
                    <div className="bg-white rounded-2xl shadow-sm border border-gray-100">
                        <div className="px-6 py-12 text-center">
                            <div className="w-14 h-14 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
                                <FaUsers className="w-6 h-6 text-gray-300" />
                            </div>
                            <p className="text-sm font-medium text-gray-500">{t('customers.noCustomers')}</p>
                            <button
                                onClick={() => setShowModal(true)}
                                className="mt-4 inline-flex items-center gap-1.5 text-sm font-semibold text-blue-600 hover:text-blue-700"
                            >
                                + {t('customers.addCustomer')}
                            </button>
                        </div>
                    </div>
                ) : (
                    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
                        <ul className="divide-y divide-gray-50">
                            {customers.map(customer => (
                                <li key={customer.id} className="flex items-center gap-4 px-6 py-4 hover:bg-gray-50 transition-colors duration-100">
                                    {/* Avatar */}
                                    <div className="w-10 h-10 rounded-full bg-gradient-to-br from-violet-500 to-violet-600 flex items-center justify-center flex-shrink-0">
                                        <span className="text-xs font-bold text-white">{getInitials(customer.name)}</span>
                                    </div>

                                    {/* Info */}
                                    <div className="flex-1 min-w-0">
                                        <h3 className="text-sm font-semibold text-gray-900 truncate">{customer.name}</h3>
                                        <div className="mt-0.5 flex flex-wrap gap-x-4 gap-y-0.5">
                                            <span className="text-xs text-gray-500 flex items-center gap-1">
                                                <FaPhone className="w-3 h-3 text-gray-400 flex-shrink-0" />
                                                {customer.phone}
                                            </span>
                                            {customer.email && (
                                                <span className="text-xs text-gray-500 flex items-center gap-1">
                                                    <FaEnvelope className="w-3 h-3 text-gray-400 flex-shrink-0" />
                                                    {customer.email}
                                                </span>
                                            )}
                                        </div>
                                        <span className="mt-0.5 text-xs text-gray-400 flex items-center gap-1">
                                            <FaMapMarkerAlt className="w-3 h-3 flex-shrink-0" />
                                            <span className="truncate">{customer.address}</span>
                                        </span>
                                        {customer.notes && (
                                            <p className="text-xs text-gray-400 mt-1 italic truncate">{customer.notes}</p>
                                        )}
                                    </div>

                                    {/* Actions */}
                                    <div className="flex items-center gap-3 flex-shrink-0">
                                        <button
                                            onClick={() => setEditingCustomer(customer)}
                                            className="text-xs font-medium text-[#1e3a5f] hover:opacity-70 transition-opacity"
                                        >
                                            {t('jobs.edit')}
                                        </button>
                                        <button
                                            onClick={() => handleDelete(customer.id)}
                                            className="text-xs font-medium text-red-500 hover:text-red-700 transition-colors"
                                        >
                                            {t('jobs.delete')}
                                        </button>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    </div>
                )}

                {showModal && (
                    <CustomerModal onSave={handleCreate} onClose={() => setShowModal(false)} />
                )}
                {editingCustomer && (
                    <CustomerModal
                        customer={editingCustomer}
                        onSave={handleUpdate}
                        onClose={() => setEditingCustomer(null)}
                    />
                )}
            </div>
        </Layout>
    );
};

export default Customers;
