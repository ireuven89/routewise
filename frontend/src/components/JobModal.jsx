import { useState } from 'react';
import { useLanguage } from '../context/LanguageContext';
import { useAuth } from '../context/AuthContext';

const JobModal = ({ job, customers, technicians, onSave, onClose }) => {
    const { t } = useLanguage();
    const { organization } = useAuth();
    const industry = organization?.industry || 'hvac';

    const [formData, setFormData] = useState({
        customer_id: job?.customer_id || '',
        technician_id: job?.technician_id || '',
        title: job?.title || '',
        description: job?.description || '',
        scheduled_at: job?.scheduled_at ? job.scheduled_at.slice(0, 16) : '',
        duration_minutes: job?.duration_minutes || 60,
        price: job?.price || '',
    });

    const handleChange = (e) => {
        const value = e.target.type === 'number' ? parseFloat(e.target.value) : e.target.value;
        setFormData({ ...formData, [e.target.name]: value });
    };

    const handleSubmit = (e) => {
        e.preventDefault();
        const jobData = {
            ...formData,
            customer_id: parseInt(formData.customer_id),
            technician_id: formData.technician_id ? parseInt(formData.technician_id) : null,
            price: formData.price ? parseFloat(formData.price) : null,
            scheduled_at: new Date(formData.scheduled_at).toISOString(),
        };
        onSave(jobData);
    };

    return (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
            <div className="bg-white rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
                <div className="px-6 py-4 border-b border-gray-200">
                    <h2 className="text-xl font-semibold text-gray-900">
                        {job ? t('jobs.editJob') : t('jobs.createNewJob')}
                    </h2>
                </div>

                <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t('jobs.customerLabel')}
                        </label>
                        <select
                            name="customer_id"
                            value={formData.customer_id}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        >
                            <option value="">{t('jobs.selectCustomer')}</option>
                            {customers.map(customer => (
                                <option key={customer.id} value={customer.id}>
                                    {customer.name} - {customer.address}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t('jobs.technicianLabel')}
                        </label>
                        <select
                            name="technician_id"
                            value={formData.technician_id}
                            onChange={handleChange}
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        >
                            <option value="">{t('jobs.unassigned')}</option>
                            {technicians.map(tech => (
                                <option key={tech.id} value={tech.id}>
                                    {tech.name}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t('jobs.jobTitle')}
                        </label>
                        <input
                            type="text"
                            name="title"
                            value={formData.title}
                            onChange={handleChange}
                            required
                            placeholder={t('jobs.jobTitlePlaceholder')}
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t('jobs.description')}
                        </label>
                        <textarea
                            name="description"
                            value={formData.description}
                            onChange={handleChange}
                            rows={3}
                            placeholder={t('jobs.descriptionPlaceholder')}
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t('jobs.scheduledDateTime')}
                        </label>
                        <input
                            type="datetime-local"
                            name="scheduled_at"
                            value={formData.scheduled_at}
                            onChange={handleChange}
                            required
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t('jobs.duration')}
                        </label>
                        <input
                            type="number"
                            name="duration_minutes"
                            value={formData.duration_minutes}
                            onChange={handleChange}
                            min="15"
                            step="15"
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-700">
                            {t('jobs.price')}
                        </label>
                        <input
                            type="number"
                            name="price"
                            value={formData.price}
                            onChange={handleChange}
                            min="0"
                            step="0.01"
                            placeholder="150.00"
                            className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                        />
                    </div>

                    <div className="flex justify-end gap-3 pt-4">
                        <button
                            type="button"
                            onClick={onClose}
                            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
                        >
                            {t('jobs.cancel')}
                        </button>
                        <button
                            type="submit"
                            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                        >
                            {job ? t('jobs.updateJob') : t('jobs.createJobBtn')}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default JobModal;
