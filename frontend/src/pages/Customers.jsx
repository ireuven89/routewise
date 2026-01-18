import {useState, useEffect, useCallback} from 'react';
import { customersAPI } from '../api/client';
import Layout from '../components/Layout';

const Customers = () => {
    const [customers, setCustomers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [editingCustomer, setEditingCustomer] = useState(null);
    const [searchTerm, setSearchTerm] = useState('');

    useEffect(() => {
        loadCustomers().catch(console.error);
    }, [loadCustomers]);

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
        void loadCustomers(); // Using void to explicitly ignore the promise
    }, [loadCustomers]);

    const handleCreate = async (customerData) => {
        try {
            await customersAPI.create(customerData);
            await loadCustomers();
            setShowModal(false);
        } catch (error) {
            console.error('Failed to create customer:', error);
            alert('Failed to create customer');
        }
    };

    const handleUpdate = async (customerData) => {
        try {
            await customersAPI.update(editingCustomer.id, customerData);
            await loadCustomers();
            setEditingCustomer(null);
        } catch (error) {
            console.error('Failed to update customer:', error);
            alert('Failed to update customer');
        }
    };

    const handleDelete = async (customerId) => {
        if (!window.confirm('Are you sure you want to delete this customer?')) return;

        try {
            await customersAPI.delete(customerId);
            await loadCustomers();
        } catch (error) {
            console.error('Failed to delete customer:', error);
            alert('Failed to delete customer');
        }
    };

    if (loading) {
        return (
            <Layout>
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">Loading customers...</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <h1 className="text-3xl font-bold text-gray-900">Customers</h1>
                    <button
                        onClick={() => setShowModal(true)}
                        className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-md font-medium"
                    >
                        + Add Customer
                    </button>
                </div>

                {/* Search */}
                <div className="mb-6">
                    <input
                        type="text"
                        placeholder="Search customers..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        onKeyUp={() => loadCustomers()}
                        className="w-full max-w-md px-4 py-2 border border-gray-300 rounded-md"
                    />
                </div>

                {/* Customers List */}
                {customers.length === 0 ? (
                    <div className="bg-white shadow rounded-lg p-8 text-center">
                        <p className="text-gray-500">No customers found. Add your first customer!</p>
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
                                        <div className="flex space-x-3">
                                            <button
                                                onClick={() => setEditingCustomer(customer)}
                                                className="text-blue-600 hover:text-blue-800 font-medium"
                                            >
                                                Edit
                                            </button>
                                            <button
                                                onClick={() => handleDelete(customer.id)}
                                                className="text-red-600 hover:text-red-800 font-medium"
                                            >
                                                Delete
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

const CustomerModal = ({ customer, onSave, onClose }) => {
    const [formData, setFormData] = useState({
        name: customer?.name || '',
        email: customer?.email || '',
        phone: customer?.phone || '',
        address: customer?.address || '',
        notes: customer?.notes || '',
    });

    const handleChange = (e) => {
        setFormData({
            ...formData,
            [e.target.name]: e.target.value,
        });
    };

    const handleSubmit = (e) => {
        e.preventDefault();
        onSave(formData);
    };

    return (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-md w-full">
                <div className="px-6 py-4 border-b border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900">
                        {customer ? 'Edit Customer' : 'Add New Customer'}
                    </h2>
                </div>

                <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Name *</label>
                        <input
                            type="text"
                            name="name"
                            value={formData.name}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-green-500 focus:border-green-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">Phone *</label>
                        <input
                            type="tel"
                            name="phone"
                            value={formData.phone}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-green-500 focus:border-green-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">Email</label>
                        <input
                            type="email"
                            name="email"
                            value={formData.email}
                            onChange={handleChange}
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-green-500 focus:border-green-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">Address *</label>
                        <input
                            type="text"
                            name="address"
                            value={formData.address}
                            onChange={handleChange}
                            required
                            placeholder="123 Main St, City, State"
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-green-500 focus:border-green-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">Notes</label>
                        <textarea
                            name="notes"
                            value={formData.notes}
                            onChange={handleChange}
                            rows={3}
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-green-500 focus:border-green-500"
                        />
                    </div>

                    <div className="flex justify-end space-x-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
                        >
                            Cancel
                        </button>
                        <button
                            type="submit"
                            className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700"
                        >
                            {customer ? 'Update' : 'Add'} Customer
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default Customers;