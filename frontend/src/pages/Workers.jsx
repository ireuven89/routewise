import { useState, useEffect } from 'react';
import { workersAPI } from '../api/client';
import Layout from '../components/Layout';

const Technicians = () => {
    const [technicians, setWorkers] = useState([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [editingTechnician, setEditingTechnician] = useState(null);
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
            console.error('Failed to load technicians:', error);
            setLoading(false);
        }
    };

    const handleCreate = async (technicianData) => {
        try {
            await workersAPI.create(technicianData);
            await loadWorkers();
            setShowModal(false);
        } catch (error) {
            console.error('Failed to create technician:', error);
            alert('Failed to create technician');
        }
    };

    const handleUpdate = async (technicianData) => {
        try {
            await workersAPI.update(editingTechnician.id, technicianData);
            await loadWorkers();
            setEditingTechnician(null);
        } catch (error) {
            console.error('Failed to update technician:', error);
            alert('Failed to update technician');
        }
    };

    const handleDelete = async (technicianId) => {
        if (!window.confirm('Are you sure you want to delete this technician?')) return;

        try {
            await workersAPI.delete(technicianId);
            await loadWorkers();
        } catch (error) {
            console.error('Failed to delete technician:', error);
            alert('Failed to delete technician');
        }
    };

    if (loading) {
        return (
            <Layout>
                <div className="flex justify-center items-center h-64">
                    <div className="text-lg text-gray-600">Loading technicians...</div>
                </div>
            </Layout>
        );
    }

    return (
        <Layout>
            <div className="px-4 sm:px-0">
                {/* Header */}
                <div className="flex justify-between items-center mb-6">
                    <h1 className="text-3xl font-bold text-gray-900">Technicians</h1>
                    <button
                        onClick={() => setShowModal(true)}
                        className="bg-purple-600 hover:bg-purple-700 text-white px-4 py-2 rounded-md font-medium"
                    >
                        + Add Technician
                    </button>
                </div>

                {/* Filter */}
                <div className="mb-6">
                    <label className="flex items-center">
                        <input
                            type="checkbox"
                            checked={showActiveOnly}
                            onChange={(e) => setShowActiveOnly(e.target.checked)}
                            className="rounded border-gray-300 text-purple-600 focus:ring-purple-500"
                        />
                        <span className="ml-2 text-sm text-gray-700">Show active only</span>
                    </label>
                </div>

                {/* Technicians List */}
                {technicians.length === 0 ? (
                    <div className="bg-white shadow rounded-lg p-8 text-center">
                        <p className="text-gray-500">No technicians found. Add your first technician!</p>
                    </div>
                ) : (
                    <div className="bg-white shadow overflow-hidden rounded-lg">
                        <ul className="divide-y divide-gray-200">
                            {technicians.map(technician => (
                                <li key={technician.id} className="px-6 py-4 hover:bg-gray-50">
                                    <div className="flex items-center justify-between">
                                        <div className="flex-1">
                                            <div className="flex items-center">
                                                <h3 className="text-lg font-medium text-gray-900">{technician.name}</h3>
                                                {technician.is_active ? (
                                                    <span className="ml-3 px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded-full">
                            Active
                          </span>
                                                ) : (
                                                    <span className="ml-3 px-2 py-1 text-xs font-medium bg-gray-100 text-gray-800 rounded-full">
                            Inactive
                          </span>
                                                )}
                                            </div>
                                            <p className="text-sm text-gray-500 mt-1">
                                                📞 {technician.phone}
                                                {technician.email && ` • ✉️ ${technician.email}`}
                                            </p>
                                        </div>
                                        <div className="flex space-x-3">
                                            <button
                                                onClick={() => setEditingTechnician(technician)}
                                                className="text-blue-600 hover:text-blue-800 font-medium"
                                            >
                                                Edit
                                            </button>
                                            <button
                                                onClick={() => handleDelete(technician.id)}
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
                    <TechnicianModal
                        onSave={handleCreate}
                        onClose={() => setShowModal(false)}
                    />
                )}

                {/* Edit Modal */}
                {editingTechnician && (
                    <TechnicianModal
                        technician={editingTechnician}
                        onSave={handleUpdate}
                        onClose={() => setEditingTechnician(null)}
                    />
                )}
            </div>
        </Layout>
    );
};

const TechnicianModal = ({ technician, onSave, onClose }) => {
    const [formData, setFormData] = useState({
        name: technician?.name || '',
        email: technician?.email || '',
        phone: technician?.phone || '',
        is_active: technician?.is_active !== undefined ? technician.is_active : true,
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
        onSave(formData);
    };

    return (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-md w-full">
                <div className="px-6 py-4 border-b border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900">
                        {technician ? 'Edit Technician' : 'Add New Technician'}
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
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-purple-500 focus:border-purple-500"
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
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-purple-500 focus:border-purple-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">Email</label>
                        <input
                            type="email"
                            name="email"
                            value={formData.email}
                            onChange={handleChange}
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-purple-500 focus:border-purple-500"
                        />
                    </div>

                    <div>
                        <label className="flex items-center">
                            <input
                                type="checkbox"
                                name="is_active"
                                checked={formData.is_active}
                                onChange={handleChange}
                                className="rounded border-gray-300 text-purple-600 focus:ring-purple-500"
                            />
                            <span className="ml-2 text-sm text-gray-700">Active</span>
                        </label>
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
                            className="px-4 py-2 bg-purple-600 text-white rounded-md hover:bg-purple-700"
                        >
                            {technician ? 'Update' : 'Add'} Technician
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default Technicians;