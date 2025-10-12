export function TableHeader() {
  return (
    <div className="grid grid-cols-[repeat(14,minmax(0,1fr))] gap-2 mb-2 text-sm font-semibold">
      <div className="col-span-2 bg-[#4A5568] text-white p-2 rounded">Name</div>
      <div className="col-span-1 bg-[#4A5568] text-white p-2 rounded text-center">
        Father Name
      </div>
      <div className="col-span-1 bg-[#4A5568] text-white p-2 rounded text-center">
        Master ID
      </div>
      <div className="col-span-1 bg-[#4A5568] text-white p-2 rounded text-center">
        ID
      </div>
      <div className="col-span-2 bg-[#9AE6B4] text-gray-900 p-2 rounded">
        Address
      </div>
      <div className="col-span-2 bg-[#FC8181] text-white p-2 rounded">
        Alt Address
      </div>
      <div className="col-span-1 bg-[#D69E2E] text-white p-2 rounded text-center">
        Email
      </div>
      <div className="col-span-1 bg-[#4299E1] text-white p-2 rounded text-center">
        Year
      </div>
      <div className="col-span-1 bg-[#ED64A6] text-white p-2 rounded text-center">
        Mobile
      </div>
      <div className="col-span-1 bg-[#805AD5] text-white p-2 rounded text-center">
        Alt Phone
      </div>
      <div className="col-span-1 bg-gray-600 text-white p-2 rounded text-center">
        Action
      </div>
    </div>
  );
}
