import {useState, useEffect, useCallback} from 'react';
import { customersAPI } from '../api/client';
import Layout from '../components/Layout';
import { useLanguage } from '../context/LanguageContext';
import CustomerModal from '../components/CustomerModal';

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
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">{t('customers.loading')}</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <h1 className="text-3xl font-bold text-gray-900">{t('customers.title')}</h1>
                    <button
                        onClick={() => setShowModal(true)}
                        className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-md font-medium"
                    >
                        {t('customers.addCustomer')}
                    </button>
                </div>

                {/* Search */}
                <div className="mb-6">
                    <input
                        type="text"
                        placeholder={t('customers.searchPlaceholder')}
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        className="w-full max-w-md px-4 py-2 border border-gray-300 rounded-md"
                    />
                </div>

                {/* Customers List */}
                {customers.length === 0 ? (
                    <div className="bg-white shadow rounded-lg p-8 text-center">
                        <p className="text-gray-500">{t('customers.noCustomers')}</p>
                    </div>
                ) : (
                    <div className="bg-white shadow overflow-hidden rounded-lg">
                        <ul className="divide-y divide-gray-200">
                            {customers.map(customer => (
                                <li key={customer.id} className="px-6 py-4 hover:bg-gray-50">
                                    <div className="flex items-center justify-between">
                                        <div className="flex-1">
                                            <h3 className="text-lg font-medium text-gray-900">{customer.name}</h3>
                                            <p className="text-sm text-gray-500 mt-1">
                                                📞 {customer.phone}
                                                {customer.email && ` • ✉️ ${customer.email}`}
                                            </p>
                                            <p className="text-sm text-gray-600 mt-1">
                                                📍 {customer.address}
                                            </p>
                                            {customer.notes && (
                                                <p className="text-sm text-gray-500 mt-2 italic">{customer.notes}</p>
                                            )}
                                        </div>
                                        <div className="flex gap-3">
                                            <button
                                                onClick={() => setEditingCustomer(customer)}
                                                className="text-blue-600 hover:text-blue-800 font-medium"
                                            >
                                                {t('jobs.edit')}
                                            </button>
                                            <button
                                                onClick={() => handleDelete(customer.id)}
                                                className="text-red-600 hover:text-red-800 font-medium"
                                            >
                                                {t('jobs.delete')}
                                            </button>
                                        </div>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    </div>
                )}

                {/* Create Modal */}
                {showModal && (
                    <CustomerModal
                        onSave={handleCreate}
                        onClose={() => setShowModal(false)}
                    />
                )}

                {/* Edit Modal */}
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
