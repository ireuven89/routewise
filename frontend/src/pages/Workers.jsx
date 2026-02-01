import { useState, useEffect } from 'react';
import { workersAPI } from '../api/client';
import Layout from '../components/Layout';
import {formatPhone} from "../utils/phone";

const COUNTRY_CODES = [
    { code: '+1', country: 'US/Canada', flag: '🇺🇸' },
    { code: '+972', country: 'Israel', flag: '🇮🇱' },
    { code: '+44', country: 'UK', flag: '🇬🇧' },
    { code: '+61', country: 'Australia', flag: '🇦🇺' },
    { code: '+91', country: 'India', flag: '🇮🇳' },
    { code: '+49', country: 'Germany', flag: '🇩🇪' },
    { code: '+33', country: 'France', flag: '🇫🇷' },
    { code: '+52', country: 'Mexico', flag: '🇲🇽' },
];

// Split stored phone "+9721234567890" into { countryCode: "+972", phoneNumber: "1234567890" }
const splitPhone = (fullPhone) => {
    if (!fullPhone) return { countryCode: '+1', phoneNumber: '' };

    const match = COUNTRY_CODES.find((c) => fullPhone.startsWith(c.code));
    if (match) {
        return {
            countryCode: match.code,
            phoneNumber: fullPhone.slice(match.code.length),
        };
    }

    return { countryCode: '+1', phoneNumber: fullPhone };
};

const Workers = () => {
    const [workers, setWorkers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [editingWorker, setEditingWorker] = useState(null);
    const [showActiveOnly, setShowActiveOnly] = useState(true);

    useEffect(() => {
        loadWorkers();
    }, [showActiveOnly]);

    const loadWorkers = async () => {
        try {
            const response = await workersAPI.getAll(showActiveOnly);
            setWorkers(response.data || []);
            setLoading(false);
        } catch (error) {
            console.error('Failed to load workers:', error);
            setLoading(false);
        }
    };

    const handleCreate = async (workerData) => {
        try {
            await workersAPI.create(workerData);
            await loadWorkers();
            setShowModal(false);
        } catch (error) {
            console.error('Failed to create worker:', error);
            alert('Failed to create worker');
        }
    };

    const handleUpdate = async (workerData) => {
        try {
            await workersAPI.update(editingWorker.id, workerData);
            await loadWorkers();
            setEditingWorker(null);
        } catch (error) {
            console.error('Failed to update worker:', error);
            alert('Failed to update worker');
        }
    };

    const handleDelete = async (workerId) => {
        if (!window.confirm('Are you sure you want to delete this worker?')) return;

        try {
            await workersAPI.delete(workerId);
            await loadWorkers();
        } catch (error) {
            console.error('Failed to delete worker:', error);
            alert('Failed to delete worker');
        }
    };

    if (loading) {
        return (
            <Layout>
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">Loading workers...</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <h1 className="text-3xl font-bold text-gray-900">Workers</h1>
                    <button
                        onClick={() => setShowModal(true)}
                        style={{ backgroundColor: '#ff6b35' }}
                        className="hover:opacity-90 text-white px-4 py-2 rounded-md font-medium"
                    >
                        + Add Worker
                    </button>
                </div>

                {/* Filter */}
                <div className="mb-6">
                    <label className="flex items-center">
                        <input
                            type="checkbox"
                            checked={showActiveOnly}
                            onChange={(e) => setShowActiveOnly(e.target.checked)}
                            className="rounded border-gray-300"
                            style={{ accentColor: '#1e3a5f' }}
                        />
                        <span className="ml-2 text-sm text-gray-700">Show active only</span>
                    </label>
                </div>

                {/* Workers List */}
                {workers.length === 0 ? (
                    <div className="bg-white shadow rounded-lg p-8 text-center">
                        <p className="text-gray-500">No workers found. Add your first worker!</p>
                    </div>
                ) : (
                    <div className="bg-white shadow overflow-hidden rounded-lg">
                        <ul className="divide-y divide-gray-200">
                            {workers.map((worker) => (
                                <li key={worker.id} className="px-6 py-4 hover:bg-gray-50">
                                    <div className="flex items-center justify-between">
                                        <div className="flex-1">
                                            <div className="flex items-center">
                                                <h3 className="text-lg font-medium text-gray-900">{worker.name}</h3>
                                                {worker.is_active ? (
                                                    <span className="ml-3 px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded-full">
                            Active
                          </span>
                                                ) : (
                                                    <span className="ml-3 px-2 py-1 text-xs font-medium bg-gray-100 text-gray-800 rounded-full">
                            Inactive
                          </span>
                                                )}
                                                {worker.role && (
                                                    <span
                                                        className="ml-3 px-2 py-1 text-xs font-medium rounded-full text-white"
                                                        style={{ backgroundColor: '#1e3a5f' }}
                                                    >
                            {worker.role}
                          </span>
                                                )}
                                            </div>
                                            <p className="text-sm text-gray-500 mt-1">
                                                📞 {worker.phone}
                                                {worker.email && ` • ✉️ ${worker.email}`}
                                            </p>
                                        </div>
                                        <div className="flex space-x-3">
                                            <button
                                                onClick={() => setEditingWorker(worker)}
                                                style={{ color: '#1e3a5f' }}
                                                className="hover:opacity-70 font-medium"
                                            >
                                                Edit
                                            </button>
                                            <button
                                                onClick={() => handleDelete(worker.id)}
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
                {showModal && <WorkerModal onSave={handleCreate} onClose={() => setShowModal(false)} />}

                {/* Edit Modal */}
                {editingWorker && (
                    <WorkerModal
                        worker={editingWorker}
                        onSave={handleUpdate}
                        onClose={() => setEditingWorker(null)}
                    />
                )}
            </div>
        </Layout>
    );
};

const WorkerModal = ({ worker, onSave, onClose }) => {
    // Split existing phone into countryCode + phoneNumber
    const { countryCode: existingCode, phoneNumber: existingNumber } = splitPhone(worker?.phone);

    const [formData, setFormData] = useState({
        name: worker?.name || '',
        email: worker?.email || '',
        countryCode: existingCode,
        phoneNumber: existingNumber,
        role: worker?.role || '',
        is_active: worker?.is_active !== undefined ? worker.is_active : true,
    });

    const handleChange = (e) => {
        const value = e.target.type === 'checkbox' ? e.target.checked : e.target.value;
        setFormData({
            ...formData,
            [e.target.name]: value,
        });
    };

    const handleSubmit = (e) => {
        e.preventDefault();

        // Combine country code + phone before sending
        const payload = {
            name: formData.name,
            email: formData.email,
            phone: formatPhone(formData.countryCode, formData.phoneNumber), // "+972" + "1234567890"
            role: formData.role,
            is_active: formData.is_active,
        };

        onSave(payload);
    };

    return (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-md w-full">
                {/* Modal Header */}
                <div
                    className="px-6 py-4 border-b border-gray-200"
                    style={{ backgroundColor: '#1e3a5f' }}
                >
                    <h2 className="text-xl font-semibold text-white">
                        {worker ? 'Edit Worker' : 'Add New Worker'}
                    </h2>
                </div>

                <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
                    {/* Name */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Name *</label>
                        <input
                            type="text"
                            name="name"
                            value={formData.name}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2"
                            style={{ focusRingColor: '#ff6b35' }}
                        />
                    </div>

                    {/* Phone with Country Code */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Phone *</label>
                        <div className="flex gap-2 mt-1">
                            {/* Country Code Dropdown */}
                            <select
                                name="countryCode"
                                value={formData.countryCode}
                                onChange={handleChange}
                                className="border border-gray-300 rounded-md px-2 py-2 bg-gray-50 focus:outline-none focus:ring-2"
                                style={{ minWidth: '100px' }}
                            >
                                {COUNTRY_CODES.map((item) => (
                                    <option key={item.code} value={item.code}>
                                        {item.flag} {item.code}
                                    </option>
                                ))}
                            </select>

                            {/* Phone Number Input */}
                            <input
                                type="tel"
                                name="phoneNumber"
                                value={formData.phoneNumber}
                                onChange={handleChange}
                                placeholder="1234567890"
                                required
                                className="flex-1 border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2"
                            />
                        </div>
                        <p className="text-xs text-gray-500 mt-1">
                            Full number: {formData.countryCode}{formData.phoneNumber}
                        </p>
                    </div>

                    {/* Email */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Email</label>
                        <input
                            type="email"
                            name="email"
                            value={formData.email}
                            onChange={handleChange}
                            className="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2"
                        />
                    </div>

                    {/* Role */}
                    <div>
                        <label className="block text-sm font-medium text-gray-700">Role</label>
                        <select
                            name="role"
                            value={formData.role}
                            onChange={handleChange}
                            className="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2 bg-white focus:outline-none focus:ring-2"
                        >
                            <option value="">Select role</option>
                            <option value="Worker">Worker</option>
                            <option value="Foreman">Foreman</option>
                            <option value="Supervisor">Supervisor</option>
                            <option value="Technician">Technician</option>
                        </select>
                    </div>

                    {/* Active Toggle */}
                    <div>
                        <label className="flex items-center">
                            <input
                                type="checkbox"
                                name="is_active"
                                checked={formData.is_active}
                                onChange={handleChange}
                                className="rounded border-gray-300"
                                style={{ accentColor: '#1e3a5f' }}
                            />
                            <span className="ml-2 text-sm text-gray-700">Active</span>
                        </label>
                    </div>

                    {/* Buttons */}
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
                            style={{ backgroundColor: '#ff6b35' }}
                            className="px-4 py-2 text-white rounded-md hover:opacity-90"
                        >
                            {worker ? 'Update' : 'Add'} Worker
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default Workers;
